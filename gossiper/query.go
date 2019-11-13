package gossiper

import (
	. "github.com/TRUMANCFY/Peerster/message"
)

type Query struct {
	id        uint32
	replyChan chan<- *SearchReply
	fileInfo  map[SHA256_HASH]*SearchFile
}

// TODO: think about what kind of data object is good to maintain the query
type SearchFile struct {
	MetafileHash SHA256_HASH
	ChunkCount   uint64
	ChunkMap     []uint64
	File         map[string]([]uint64)
}

func (s *SearchHandler) WatchNewQuery(keywords []string) *Query {
	// TODO: Generate new query
	q := &Query{}
	return q
}
