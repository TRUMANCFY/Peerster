package gossiper

import (
	"fmt"

	. "github.com/TRUMANCFY/Peerster/message"
)

// THIS FILE FOR THE HOMEWORK2
func (g *Gossiper) updateRouteTable(newRumor *RumorMessage, senderAddrStr string) {
	// because we have checked the diff

	// lock
	g.routeTable.Mux.Lock()
	defer g.routeTable.Mux.Unlock()

	origin := newRumor.Origin
	oldVal, present := g.routeTable.routeTable[origin]

	g.routeTable.routeTable[origin] = senderAddrStr

	// OUTPUT
	if !present || oldVal != senderAddrStr {

		if newRumor.Text != "" {
			fmt.Printf("DSDV %s %s \n", origin, senderAddrStr)
		}
	}
}
