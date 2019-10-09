## Code Structure for Homework 1
### Data Structure
1. Self peerStatus: A map containing the local message status ```map[string]PeerStatus```

2. Peer's peerStatus: ```map[string](map[string]PeerStatus) ```
3. Rumour List: the rumours contained: ``` map[string](map[sequenceNumber]RumorMessage) ```
4. Simple List: the simple messages contained: ``` map[string][]SimpleMessage ```


### Event Handler
