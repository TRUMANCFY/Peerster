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

const START_BUDGET = 2
const MAX_BUDGET = 32
const EXP_SLEEP_TIME = 1 * time.Second

type TaskDistribution struct {
	Peer   string
	Budget uint64
}

type SearchHandler struct {
	Name           string
	searchRecords  *SearchRecords
	currentQueryID *QueryID
}

type SearchRecords struct {
	SearchHistory map[string](map[string]time.Time) // map[Origin][keywordsStr]recordTime
	Mux           *sync.Mutex
}

func NewSearchHandler(name string) *SearchHandler {
	searchHistory := make(map[string](map[string]time.Time))
	searchRecords := &SearchRecords{
		SearchHistory: searchHistory,
		Mux:           &sync.Mutex{},
	}

	queryId := &QueryID{
		id:  0,
		Mux: &sync.Mutex{},
	}

	searchHandler := &SearchHandler{
		searchRecords:  searchRecords,
		Name:           name,
		currentQueryID: queryId,
	}

	return searchHandler
}

func (g *Gossiper) HandleClientSearch(cmw *ClientMessageWrapper) {
	// we have already checked in the client operation that
	keywordsStr := *cmw.msg.Keywords
	fmt.Println(keywordsStr)
	budget := *cmw.msg.Budget
	fmt.Println(budget)

	keywords := strings.Split(keywordsStr, ",")

	query := g.fileHandler.WatchNewQuery(keywords)

	if budget != 0 {
		fmt.Println("Common search with budget ", budget)

		searchRequest := &SearchRequest{
			Origin:   g.name,
			Budget:   budget,
			Keywords: keywords,
		}

		// TODO: In this case, if the budget is not enough just leave the query channel here?????
		g.HandleSearchRequest(searchRequest, nil)
	} else {
		// TODO: expotenial
		budget = 2

		for !(query.isDone() || budget > MAX_BUDGET) {
			fmt.Println("EXP BUDGET ", budget)
			searchRequest := &SearchRequest{
				Origin:   g.name,
				Budget:   budget,
				Keywords: keywords,
			}

			go g.HandleSearchRequest(searchRequest, nil)

			time.Sleep(EXP_SLEEP_TIME)
			budget *= 2
		}
	}
}

func (g *Gossiper) HandleSearchRequest(searchRequest *SearchRequest, sender *net.UDPAddr) {
	// check whether the search request has 0.5 second later
	valid := g.fileHandler.searchHandler.checkDuplicate(searchRequest)

	fmt.Println("Handle Search Request")

	if !valid {
		return
	}

	// forward request to the neighbors
	go g.DistributeSearchRequest(searchRequest, sender)

	// local search
	g.LocalSearch(searchRequest)
}

func (g *Gossiper) HandleSearchReply(searchReply *SearchReply, sender *net.UDPAddr) {
	if DEBUGSEARCH {
		fmt.Printf("Receive Search Reply from %s to %s \n", searchReply.Origin, searchReply.Destination)
	}

	if searchReply.Destination == g.name {
		// TODO: accept the reply
		g.fileHandler.searchDispatcher.searchReplyChan <- searchReply
		return
	}

	reply, valid := g.fileHandler.searchHandler.prepareNewReply(searchReply)

	if !valid {
		return
	}

	success := g.RouteSearchReply(reply)

	if !success {
		if DEBUGSEARCH {
			fmt.Println("Route Search Reply Fails")
		}
	}

}

func (g *Gossiper) LocalSearch(searchRequest *SearchRequest) {
	searchedFiles := g.fileHandler.SearchFileKeywords(searchRequest.Keywords)

	searchReply := g.fileHandler.searchHandler.GenerateSearchReply(searchedFiles, searchRequest.Origin)

	g.RouteSearchReply(searchReply)
}

func (g *Gossiper) RouteSearchReply(searchReply *SearchReply) bool {
	dest := searchReply.Destination

	g.routeTable.Mux.Lock()
	nextNode, present := g.routeTable.routeTable[dest]
	g.routeTable.Mux.Unlock()

	if !present {
		if DEBUGFILE {
			fmt.Println("Destination %s does not exist in the table \n", dest)
		}
		return false
	}

	if DEBUGFILE {
		fmt.Printf("Send the searchReply Dest: %s to Nextnode %s \n", dest, nextNode)
	}

	g.SendGossipPacketStrAddr(&GossipPacket{SearchReply: searchReply}, nextNode)

	return true
}

func (f *FileHandler) SearchFileKeywords(keywords []string) []*File {

	fmt.Println("Search ", strings.Join(keywords, ","))
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

	fmt.Println(searchedFile)

	return searchedFile
}

func (s *SearchHandler) GenerateSearchResult(searchedFiles []*File) []*SearchResult {
	searchResults := make([]*SearchResult, 0)

	for _, f := range searchedFiles {
		searchR := &SearchResult{
			FileName:     f.Name,
			MetafileHash: f.MetafileHash[:],
			ChunkMap:     f.ChunkMap,
			ChunkCount:   f.ChunkCount,
		}

		searchResults = append(searchResults, searchR)
	}

	return searchResults
}

func (s *SearchHandler) GenerateSearchReply(searchedFiles []*File, dest string) *SearchReply {
	searchResult := s.GenerateSearchResult(searchedFiles)

	searchReply := &SearchReply{
		Origin:      s.Name,
		Destination: dest,
		HopLimit:    HOPLIMIT,
		Results:     searchResult,
	}
	return searchReply
}

func (s *SearchHandler) checkDuplicate(searchRequest *SearchRequest) bool {
	// we need to check whether it has been shown in previous 0.5 second
	s.searchRecords.Mux.Lock()
	defer s.searchRecords.Mux.Unlock()

	origin := searchRequest.Origin
	keywords := searchRequest.Keywords

	// sort the str first
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

	for _, td := range taskDistribution {
		fmt.Printf("Peer: %s Budget: %d \n", td.Peer, td.Budget)
	}

	g.SpreadSearchRequest(searchRequest, taskDistribution)
}

func (g *Gossiper) DistributeBudget(budget uint64, sender *net.UDPAddr) []TaskDistribution {
	taskDistribution := make([]TaskDistribution, 0)

	var senderStr string
	if sender != nil {
		senderStr = sender.String()
	} else {
		senderStr = ""
	}

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
	leftInt := int(budget) % numNeighbor

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
		// Attention: sending a task with budget == 0 is dangerous
		// for an unsigned int 18446744073709551615 == -1
		if task.Budget == 0 {
			continue
		}
		sReq := &SearchRequest{
			Origin:   searchRequest.Origin,
			Budget:   task.Budget,
			Keywords: searchRequest.Keywords,
		}
		go g.SendGossipPacketStrAddr(&GossipPacket{SearchRequest: sReq}, task.Peer)
	}
}

func (s *SearchHandler) prepareNewReply(searchReply *SearchReply) (*SearchReply, bool) {
	if searchReply.HopLimit == 0 {
		if DEBUGSEARCH {
			fmt.Println("HopLimit has been ended")
		}
		return nil, false
	}

	newSearchReply := &SearchReply{
		Origin:      searchReply.Origin,
		Destination: searchReply.Destination,
		HopLimit:    searchReply.HopLimit - 1,
		Results:     searchReply.Results,
	}

	return newSearchReply, true
}
