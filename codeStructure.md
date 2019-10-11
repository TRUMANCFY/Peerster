# Code Structure for Homework 1
## Data Structure
1. Self peerStatus: A map containing the local message status ```map[string]PeerStatus```

2. Peer's peerStatus: ```map[string](map[string]PeerStatus) ```
3. Rumour List: the rumours contained: ``` map[string](map[sequenceNumber]RumorMessage) ```
4. Simple List: the simple messages contained: ``` map[string][]SimpleMessage ```
5. Gossip has a toSendChan to dispatch the progress of sending out.


## Event Handler
#### Gossiper
##### Finished
- ``` func (g *Gossiper) Run() ```:
Gossiper start to run
- ``` func (g *Gossiper) Listen(peerListener <-chan *GossipPacketWrapper, clientListener <-chan *ClientMessageWrapper)  ```
	- Gossiper listens to the gossipPacket and the clientMessage
- ```func (g *Gossiper) ReceiveFromPeers() <-chan *GossipPacketWrapper ```
	- Receive Packet from Peers, decode the message, and return a channel
- ```func (g *Gossiper) ReceiveFromClients() <-chan *ClientMessageWrapper```
	- Receive Message from Clients, decode the message, and return a channel
- ``` func MessageReceive(conn *net.UDPConn) <-chan *MessageReceived ```
	- Receive from the specific connection
- ``` func (g *Gossiper) HandlePeerMessage(gpw *GossipPacketWrapper) ```
	-  Deal with the Peer Message, figure out the type of the Message
- ``` func (g *Gossiper) HandleClientMessage(cmw *ClientMessageWrapper) ```
	- Deal with the client message. Based on the type of message (simple or rumor), we can decide the following handler
- ``` func (g *Gossiper) RumorStatusCheck(r *RumorMessage) int ```
	- Check the relationship between the rumour status received and local rumour status

- ``` func (g *Gossiper) SendGossipPacket(gp *GossipPacket, target *net.UDPAddr)```
	- Send the GossipPacket to some address
- ```func (g *Gossiper) SendGossipPacketStrAddr(gp *GossipPacket, targetStr string)```
	- the address is _String_
- ``` func (g *Gossiper) SendClientAck(client *net.UDPAddr) ```
	- Send a message object containing text "Ok" to the client 
- ``` func (g *Gossiper) BroadcastPacket(gp *GossipPacket, excludedPeers *StringSet) ```
	- Broadcast the gossip packet to all peers, except the _excludedPeers_
- ``` func (g *Gossiper) SendPacket(gp *GossipPacket, peerAddr string) ```
	- <span style="color:red"> Should be abort, as we have already had _toSendChan_</span>
- ``` func (g *Gossiper) SelectRandomNeighbor(excludedPeer *StringSet) (string, bool) ```
-  *????* ``` func (g *Gossiper) HandleStatusPacket(s *StatusPacket) ```

#### Deprecated
- ```func (g *Gossiper) CreateForwardPacket(m *SimpleMessage) *SimpleMessage```
	- Create simple message packet for transmission to the next node
- 

##### Not Finished
- ```func (g *Gossiper) HandleSimplePacket(s *SimpleMessage) ```
- ```func (g *Gossiper) HandleStatusPacket(s *StatusPacket)```
- ``` func (g *Gossiper) HandleRumorPacket(r *RumorMessage, senderAddr *net.UDPAddr) ```
- AcceptRumor
	- Add the new rumour in the rumour list
	- update the 


