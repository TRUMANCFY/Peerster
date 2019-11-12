package gossiper

import (
	"fmt"
	"math/rand"
	"net"
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
	SearchHistory map[string](map[string]time.Time) // map[Origin][]
	Mux           *sync.Mutex
}

func NewSearchHandler() *SearchHandler {
	// searchHistory := make(map[string])
}

func (g *Gossiper) HandleSearchRequest(searchRequest *SearchRequest, sender *net.UDPAddr) {

}

func (g *Gossiper) HandleSearchReply(searchReply *SearchReply, sender *net.UDPAddr) {

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
