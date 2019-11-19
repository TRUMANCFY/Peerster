package gossiper

import (
	"fmt"
	"strings"
	"sync"

	. "github.com/TRUMANCFY/Peerster/message"
)

const MATCH_THRESHOLD = 2

type Query struct {
	id        uint32
	replyChan chan<- *SearchReply
	fileInfo  map[SHA256_HASH]*SearchFile
	keywords  []string
	qLock     *sync.Mutex
	Result    []SHA256_HASH
}

// TODO: think about what kind of data object is good to maintain the query
type SearchFile struct {
	MetafileHash  SHA256_HASH
	FileName      string
	ChunkCount    uint64
	TotalChunkMap []uint64
	chunkSrc      map[string]*ChunkSource
}

type ChunkSource struct {
	Origin   string
	ChunkMap []uint64
}

func (f *FileHandler) WatchNewQuery(keywords []string) *Query {
	replyChan := make(chan *SearchReply, CHANNEL_BUFFER_SIZE)
	fileInfo := make(map[SHA256_HASH]*SearchFile)
	f.searchHandler.currentQueryID.Mux.Lock()

	q := &Query{
		id:        f.searchHandler.currentQueryID.id,
		replyChan: replyChan,
		fileInfo:  fileInfo,
		keywords:  keywords,
		qLock:     &sync.Mutex{},
		Result:    make([]SHA256_HASH, 0),
	}

	// increase the current query ID
	f.searchHandler.currentQueryID.id++

	f.searchHandler.currentQueryID.Mux.Unlock()

	// register first
	f.RegisterQuery(q)

	go func() {
		defer f.UnregisterQuery(q)

		for {
			searchReply := <-replyChan

			// firstly check whether this reply is our needed
			// if needed, put it in our data structure

			fmt.Printf("Receive reply fromm %s with %d results \n", searchReply.Origin, len(searchReply.Results))

			q.matchWithKeywords(searchReply)

			if q.isDone() {
				fmt.Println("SEARCH FINISH")
				return
			}
		}
	}()

	return q
}

func (q *Query) matchWithKeywords(searchReply *SearchReply) {
	searchResult := searchReply.Results
	// check whether this data reply contains what we need
	q.qLock.Lock()
	defer q.qLock.Unlock()

	for _, kw := range q.keywords {
		for _, sr := range searchResult {
			fmt.Println(sr.FileName)
			if strings.Contains(sr.FileName, kw) {
				chunkStr := make([]string, len(sr.ChunkMap))
				for i, c := range sr.ChunkMap {
					chunkStr[i] = fmt.Sprintf("%d", c)
				}

				// Print std output
				fmt.Printf("FOUND match %s at %s metafile=%x chunks=%s \n", sr.FileName, searchReply.Origin, sr.MetafileHash, strings.Join(chunkStr, ","))
				// check the query whether it already know about this information
				sha, _ := HashToSha256(sr.MetafileHash)
				searchFile, present := q.fileInfo[sha]

				// update
				if !present {

					chunkSrc := make(map[string]*ChunkSource, 0)

					chunkSrc[searchReply.Origin] = &ChunkSource{
						Origin:   searchReply.Origin,
						ChunkMap: sr.ChunkMap,
					}

					totalChunkMap := make([]uint64, sr.ChunkCount)

					// init with 0
					for ind := range totalChunkMap {
						totalChunkMap[ind] = 0
					}

					// init with sr chunk [count start from 1]
					for _, ind := range sr.ChunkMap {
						totalChunkMap[ind-1] = uint64(1)
					}

					q.fileInfo[sha] = &SearchFile{
						MetafileHash:  sha,
						FileName:      sr.FileName,
						ChunkCount:    sr.ChunkCount,
						TotalChunkMap: totalChunkMap,
						chunkSrc:      chunkSrc,
					}
				} else {
					// the file exist
					// TODO: if two hash file have different filename as they come from different origin
					if searchFile.ChunkCount != sr.ChunkCount {
						panic("Chunk Count does not match with previous count!")
					}

					// update the totalChunkMap
					for _, ind := range sr.ChunkMap {
						q.fileInfo[sha].TotalChunkMap[ind-1] = uint64(1)
					}
					// TODO: think about whether we should update here????
					// if what we received is the same file with the same origin
					_, present := searchFile.chunkSrc[searchReply.Origin]

					if !present {
						q.fileInfo[sha].chunkSrc[searchReply.Origin] = &ChunkSource{
							Origin:   searchReply.Origin,
							ChunkMap: sr.ChunkMap,
						}
					}

				}
			}
		}
	}

	q.GenerateDownloadList()
}

func (q *Query) GenerateDownloadList() {
	downloadList := make([]SHA256_HASH, 0)

	for sha, sf := range q.fileInfo {
		existingChunkNum := sum(sf.TotalChunkMap)

		// fulfill the requirement
		if existingChunkNum == sf.ChunkCount {
			downloadList = append(downloadList, sha)
		}
	}

	q.Result = downloadList
}

func (q *Query) isDone() bool {
	if len(q.Result) >= MATCH_THRESHOLD {
		return true
	}
	return false
}

func sum(l []uint64) uint64 {
	res := uint64(0)
	for _, ele := range l {
		res += ele
	}

	return res
}
