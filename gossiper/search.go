package gossiper

import (
	"crypto/sha256"
	"fmt"
	"net"
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

		go func() {
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
		}()
	}
}

func (g *Gossiper) HandleSearchRequest(searchRequest *SearchRequest, sender *net.UDPAddr) {
	if DEBUGSEARCH {
		fmt.Printf("Received Search Request from %s with Budget %d Origin %s \n", sender.String(), searchRequest.Budget, searchRequest.Origin)
	}
	// check whether the search request has 0.5 second later
	valid := g.fileHandler.searchHandler.checkDuplicate(searchRequest)

	fmt.Println("Handle Search Request")

	if !valid {
		return
	}

	// forward request to the neighbors
	g.DistributeSearchRequest(searchRequest, sender)

	// local search (and if the node is sender, no need for local search)
	if sender != nil {
		g.LocalSearch(searchRequest)
	}
}

func (g *Gossiper) HandleSearchReply(searchReply *SearchReply, sender *net.UDPAddr) {
	if DEBUGSEARCH {
		fmt.Printf("Receive Search Reply from %s to %s \n", searchReply.Origin, searchReply.Destination)
	}

	if searchReply.Destination == g.name {
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
	if DEBUGSEARCH {
		fmt.Printf("Start Local Search for keywords: %s \n", strings.Join(searchRequest.Keywords, ","))
	}

	searchedFiles := g.fileHandler.SearchFileKeywords(searchRequest.Keywords)

	if DEBUGSEARCH {
		fmt.Printf("Found %d files \n", len(searchedFiles))
	}

	if len(searchedFiles) == 0 {
		return
	}

	searchReply := g.fileHandler.GenerateSearchReply(searchedFiles, searchRequest.Origin)

	if DEBUGSEARCH {
		fmt.Printf("Generate Reply to %s with %d results \n", searchReply.Destination, len(searchReply.Results))
	}

	g.RouteSearchReply(searchReply)

}

func (g *Gossiper) RouteSearchReply(searchReply *SearchReply) bool {
	dest := searchReply.Destination

	g.routeTable.Mux.Lock()
	nextNode, present := g.routeTable.routeTable[dest]
	g.routeTable.Mux.Unlock()

	if !present {
		if DEBUGSEARCH {
			fmt.Println("Destination %s does not exist in the table \n", dest)
		}
		return false
	}

	if DEBUGSEARCH {
		fmt.Printf("Send the searchReply Dest: %s to Nextnode %s \n", dest, nextNode)
	}

	g.SendGossipPacketStrAddr(&GossipPacket{SearchReply: searchReply}, nextNode)

	return true
}

func (f *FileHandler) SearchFileKeywords(keywords []string) []*File {

	fmt.Println("Search ", strings.Join(keywords, ","))

	flag := false
	searchedFile := make([]*File, 0)

	f.filesLock.Lock()

	for _, file := range f.files {
		flag = false
		for _, kw := range keywords {
			if strings.Contains(file.Name, kw) {
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

func (fh *FileHandler) GenerateSearchResult(searchedFiles []*File) []*SearchResult {
	searchResults := make([]*SearchResult, 0)

	for _, f := range searchedFiles {
		// searchR := &SearchResult{
		// 	FileName:     f.Name,
		// 	MetafileHash: f.MetafileHash[:],
		// 	ChunkMap:     f.ChunkMap,
		// 	ChunkCount:   f.ChunkCount,
		// }

		// generate chunkMap and chunkCount

		chunkMap := fh.chunkMap(f)
		chunkCount := f.chunkCount()

		searchR := &SearchResult{
			FileName:     f.Name,
			MetafileHash: f.MetafileHash[:],
			ChunkMap:     chunkMap,
			ChunkCount:   chunkCount,
		}

		searchResults = append(searchResults, searchR)
	}

	if DEBUGSEARCH {
		fmt.Printf("Found %d results \n", len(searchResults))
	}

	return searchResults
}

func (fh *FileHandler) chunkMap(f *File) []uint64 {
	chunkMap := make([]uint64, 0)
	numChunks := len(f.Metafile) / sha256.Size

	for i := 0; i < numChunks; i++ {
		metaChunk, _ := HashToSha256(f.Metafile[i*sha256.Size : (i+1)*sha256.Size])
		fh.fileChunksLock.Lock()
		if _, present := fh.fileChunks[metaChunk]; present {
			chunkMap = append(chunkMap, uint64(i+1))
		}
		fh.fileChunksLock.Unlock()
	}

	if DEBUGSEARCH {
		fmt.Printf("File %s with %d chunks \n", f.Name, len(chunkMap))
	}

	return chunkMap
}

func (f *File) chunkCount() uint64 {
	return uint64(len(f.Metafile) / sha256.Size)
}

func (fh *FileHandler) GenerateSearchReply(searchedFiles []*File, dest string) *SearchReply {
	searchResult := fh.GenerateSearchResult(searchedFiles)

	searchReply := &SearchReply{
		Origin:      fh.Name,
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
	baseInt := budget / uint64(numNeighbor)
	leftInt := budget % uint64(numNeighbor)

	budgetList := make([]uint64, numNeighbor)

	for ind := range budgetList {
		if uint64(ind) < leftInt {
			budgetList[ind] = baseInt + 1
		} else {
			budgetList[ind] = baseInt
		}
	}

	// shuffle the neighbors
	// rand.Shuffle(numNeighbor, func(i, j int) { neighbors[i], neighbors[j] = neighbors[j], neighbors[i] })

	fmt.Println("Total Budget is ", budget)

	for ind := range neighbors {
		task := TaskDistribution{
			Peer:   neighbors[ind],
			Budget: budgetList[ind],
		}

		if DEBUGSEARCH {
			fmt.Printf("=== Peer Target: %s; Budget: %d === \n", task.Peer, task.Budget)
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
		Destination: searchReply.Destination,
		HopLimit:    searchReply.HopLimit - 1,
		Origin:      searchReply.Origin,
		Results:     searchReply.Results,
	}

	return newSearchReply, true
}
