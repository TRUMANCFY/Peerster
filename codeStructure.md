# Code Structure for Homework 1
## Data Structure
1. Self peerStatus: A map containing the local message status ```map[string]PeerStatus```

2. Peer's peerStatus: ```map[string](map[string]PeerStatus) ```
3. Rumour List: the rumours contained: ``` map[string](map[sequenceNumber]RumorMessage) ```
4. Simple List: the simple messages contained: ``` map[string][]SimpleMessage ```


## Event Handler
#### Gossiper
##### Finished
- ``` func (g *Gossiper) Run() ```
Gossiper start to run
- ``` func (g *Gossiper) Listen(peerListener <-chan *GossipPacketWrapper, clientListener <-chan *ClientMessageWrapper)  ```
- ```func (g *Gossiper) ReceiveFromPeers() <-chan *GossipPacketWrapper ```
- ```func (g *Gossiper) ReceiveFromClients() <-chan *ClientMessageWrapper``` 
- ``` func (g *Gossiper) HandlePeerMessage(gpw *GossipPacketWrapper) ```
- ``` func (g *Gossiper) HandleClientMessage(cmw *ClientMessageWrapper) ```

- ``` func (g *Gossiper) HandleStatusPacket(s *StatusPacket) ```
- ``` func (g *Gossiper) RumorStatusCheck(r *RumorMessage) int ```
- ``` func (g *Gossiper) ReceiveFromPeers() <-chan *GossipPacketWrapper ```
- ```func (g *Gossiper) ReceiveFromClients() <-chan *ClientMessageWrapper```
- ``` func MessageReceive(conn *net.UDPConn) <-chan *MessageReceived ```
- ```func (g *Gossiper) CreateForwardPacket(m *SimpleMessage) *SimpleMessage```
- ```func (g *Gossiper) CreateForwardPacket(m *SimpleMessage) *SimpleMessage```
- 

##### Not Finished
- ```func (g *Gossiper) HandleSimplePacket(s *SimpleMessage) ```
- ```func (g *Gossiper) HandleStatusPacket(s *StatusPacket)```
- ``` func (g *Gossiper) HandleRumorPacket(r *RumorMessage, senderAddr *net.UDPAddr) ```


