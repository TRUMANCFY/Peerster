package gossiper

import (
	"fmt"
	"math/rand"
	"net"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
)

type TaskDistribution struct {
	Peer   string
	Budget uint64
}

type SearchHandler struct {
	searchRecords *SearchRecords
}

type SearchRecords struct {
	SearchHistory map[string](map[string]time.Time) // map[Origin][keywordsStr]recordTime
	Mux           *sync.Mutex
}

func NewSearchHandler() *SearchHandler {
	searchHistory := make(map[string](map[string]time.Time))
	searchRecords := &SearchRecords{
		SearchHistory: searchHistory,
		Mux:           &sync.Mutex{},
	}

	searchHandler := &SearchHandler{
		searchRecords: searchRecords,
	}

	return searchHandler
}

func (g *Gossiper) HandleSearchRequest(searchRequest *SearchRequest, sender *net.UDPAddr) {
	// check whether the search request has 0.5 second later
	valid := g.searchHandler.checkDuplicate(searchRequest)

	if !valid {
		return
	}

	go g.DistributeSearchRequest(searchRequest, sender)

	// local search
	g.LocalSearch(searchRequest)
}

func (g *Gossiper) HandleSearchReply(searchReply *SearchReply, sender *net.UDPAddr) {

}

func (g *Gossiper) LocalSearch(searchRequest *SearchRequest) {
	searchedFiles := g.fileHandler.SearchFileKeywords(searchRequest.Keywords)

	fmt.Println(searchedFiles)

}

func (f *FileHandler) SearchFileKeywords(keywords []string) []*File {
	regExpList := make([]*regexp.Regexp, 0)

	// have a list of reg
	for _, kw := range keywords {
		kwRegExp, _ := regexp.Compile(kw)
		regExpList = append(regExpList, kwRegExp)
	}

	flag := false
	searchedFile := make([]*File, 0)

	f.filesLock.Lock()

	for _, file := range f.files {
		flag = false
		for _, re := range regExpList {
			matchedStr := re.FindString(file.Name)
			if matchedStr != "" {
				flag = true
			}
		}

		if flag {
			searchedFile = append(searchedFile, file)
		}
	}

	f.filesLock.Unlock()

	return searchedFile

}

func (s *SearchHandler) checkDuplicate(searchRequest *SearchRequest) bool {
	// we need to check whether it has been shown in previous 0.5 second
	s.searchRecords.Mux.Lock()
	defer s.searchRecords.Mux.Unlock()

	origin := searchRequest.Origin
	keywords := searchRequest.Keywords

	sort.Strings(keywords)

	keywordsStr := strings.Join(keywords, "")

	_, present := s.searchRecords.SearchHistory[origin]

	if !present {
		s.searchRecords.SearchHistory[origin] = make(map[string]time.Time)
	}

	prevTime, present := s.searchRecords.SearchHistory[origin][keywordsStr]

	// update the time
	nowTime := time.Now()
	s.searchRecords.SearchHistory[origin][keywordsStr] = nowTime

	if !present {
		return true
	}

	// get the currentTime
	timeDiff := int64(nowTime.Sub(prevTime) / time.Millisecond)

	if timeDiff > 500 {
		return true
	} else {
		return false
	}

	return false
}

func (g *Gossiper) DistributeSearchRequest(searchRequest *SearchRequest, sender *net.UDPAddr) {
	if searchRequest.Budget == 1 {
		if DEBUGSEARCH {
			fmt.Println("No enough budget to devide")
		}
		return
	}

	// the budget is enough to redistribute
	// the budget will be decreased by 1
	taskDistribution := g.DistributeBudget(searchRequest.Budget-1, sender)

	g.SpreadSearchRequest(searchRequest, taskDistribution)
}

func (g *Gossiper) DistributeBudget(budget uint64, sender *net.UDPAddr) []TaskDistribution {
	taskDistribution := make([]TaskDistribution, 0)

	senderStr := sender.String()

	// divide the task first
	g.peersList.Mux.Lock()
	peers := g.peersList.PeersList.ToArray()
	g.peersList.Mux.Unlock()

	// the peers are empty
	if len(peers) == 0 {
		if DEBUGSEARCH {
			fmt.Println("There is no neighbors for us to choose")
		}
		return taskDistribution
	}

	// just one peer and it is the same as the source
	if len(peers) == 1 && peers[0] == senderStr {
		if DEBUGSEARCH {
			fmt.Println("We do have one neighbor and it is exactly the source")
		}
		return taskDistribution
	}

	// get the neighbor list
	var neighbors []string

	for _, peer := range peers {
		if peer != senderStr {
			neighbors = append(neighbors, peer)
		}
	}

	numNeighbor := len(neighbors)
	baseInt := int(budget) / numNeighbor
	leftInt := int(budget) - baseInt*numNeighbor

	budgetList := make([]uint64, numNeighbor)

	for ind := range budgetList {
		if ind < leftInt {
			budgetList[ind] = uint64(baseInt + 1)
		} else {
			budgetList[ind] = uint64(baseInt)
		}
	}

	// shuffle the neighbors
	rand.Shuffle(numNeighbor, func(i, j int) { neighbors[i], neighbors[j] = neighbors[j], neighbors[i] })

	for ind := range neighbors {
		task := TaskDistribution{
			Peer:   neighbors[ind],
			Budget: budgetList[ind],
		}
		taskDistribution = append(taskDistribution, task)
	}

	return taskDistribution
}

func (g *Gossiper) SpreadSearchRequest(searchRequest *SearchRequest, tasks []TaskDistribution) {
	for _, task := range tasks {
		sReq := &SearchRequest{
			Origin:   searchRequest.Origin,
			Budget:   task.Budget,
			Keywords: searchRequest.Keywords,
		}
		g.SendGossipPacketStrAddr(&GossipPacket{SearchRequest: sReq}, task.Peer)
	}
}
