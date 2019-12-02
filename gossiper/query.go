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
	FullMatch int
}

// TODO: think about what kind of data object is good to maintain the query
type SearchFile struct {
	MetafileHash  SHA256_HASH
	ChunkCount    uint64
	TotalChunkMap []uint64
	chunkSrc      map[string]*ChunkSource
}

type ChunkSource struct {
	Origin   string
	ChunkMap []uint64
	Filename string
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
		FullMatch: 0,
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

			if DEBUGSEARCH {
				fmt.Printf("Receive reply fromm %s with %d results \n", searchReply.Origin, len(searchReply.Results))
			}

			q.matchWithKeywords(searchReply)

			if q.isDone() {
				if HW3OUTPUT {
					fmt.Println("SEARCH FINISHED")
				}

				f.searchFiles.Mux.Lock()
				for _, sha := range q.Result {
					q.qLock.Lock()
					f.searchFiles.searchedFiles[sha] = q.fileInfo[sha]
					q.qLock.Unlock()
				}
				f.searchFiles.Mux.Unlock()
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

	for _, kw := range q.keywords {
		for _, sr := range searchResult {
			if DEBUGSEARCH {
				fmt.Println(sr.FileName)
			}

			if strings.Contains(sr.FileName, kw) {
				chunkStr := make([]string, len(sr.ChunkMap))
				for i, c := range sr.ChunkMap {
					chunkStr[i] = fmt.Sprintf("%d", c)
				}

				// Print std output
				if HW3OUTPUT {
					fmt.Printf("FOUND match %s at %s metafile=%x chunks=%s \n", sr.FileName, searchReply.Origin, sr.MetafileHash, strings.Join(chunkStr, ","))
					// fmt.Printf("DOWNLOADING metafile of %s from %s \n", sr.FileName, searchReply.Origin)
				}
				// check the query whether it already know about this information
				sha, _ := HashToSha256(sr.MetafileHash)
				searchFile, present := q.fileInfo[sha]

				// update
				if !present {

					chunkSrc := make(map[string]*ChunkSource, 0)

					chunkSrc[searchReply.Origin] = &ChunkSource{
						Origin:   searchReply.Origin,
						ChunkMap: sr.ChunkMap,
						Filename: sr.FileName,
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
							Filename: sr.FileName,
						}
					}

				}
			}
		}
	}

	q.qLock.Unlock()

	q.GenerateDownloadList()
}

func (q *Query) GenerateDownloadList() {
	downloadList := make([]SHA256_HASH, 0)

	// HASH - FILENAME - NODE
	shaFilenameRecord := make(map[SHA256_HASH](map[string](map[string]bool)))

	for sha, sf := range q.fileInfo {
		// outdated
		// existingChunkNum := sum(sf.TotalChunkMap)

		// // fulfill the requirement
		// if existingChunkNum == sf.ChunkCount {
		// 	downloadList = append(downloadList, sha)
		// }

		// the current one is (hash, filename) all the chunks in the one node
		for _, src := range sf.chunkSrc {
			if uint64(len(src.ChunkMap)) == sf.ChunkCount {
				// this is a full match
				_, present := shaFilenameRecord[sha]
				if !present {
					shaFilenameRecord[sha] = make(map[string](map[string]bool))
					shaFilenameRecord[sha][src.Filename] = make(map[string]bool)
					shaFilenameRecord[sha][src.Filename][src.Origin] = true
				} else {
					_, present = shaFilenameRecord[sha][src.Filename]
					if !present {
						shaFilenameRecord[sha][src.Filename] = make(map[string]bool)
					}
					shaFilenameRecord[sha][src.Filename][src.Origin] = true
				}
			}
		}
	}
	// get the number first
	num := 0
	for sha, filenameOriginMap := range shaFilenameRecord {
		downloadList = append(downloadList, sha)
		for _, originMap := range filenameOriginMap {
			for _, _ = range originMap {
				num++
			}
		}
	}

	q.qLock.Lock()
	q.Result = downloadList
	q.FullMatch = num
	q.qLock.Unlock()

}

func (q *Query) isDone() bool {
	// if len(q.Result) >= MATCH_THRESHOLD {
	// 	return true
	// }
	if q.FullMatch >= MATCH_THRESHOLD {
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
