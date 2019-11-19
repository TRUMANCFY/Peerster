package gossiper

import "fmt"

func (g *Gossiper) RequestSearchedFile(metafileHash SHA256_HASH, localFileName string) bool {
	// check whther it already exist local
	g.fileHandler.filesLock.Lock()
	lookup, present := g.fileHandler.files[metafileHash]
	g.fileHandler.filesLock.Unlock()

	if present {
		if lookup.State == Shared {
			if DEBUGFILE {
				fmt.Println("File is local shared file")
			}
			return true
		} else if lookup.State == Downloaded {
			if DEBUGFILE {
				fmt.Println("File has been already downloaded")
			}
			return true
		} else if lookup.State == Downloading {
			if DEBUGFILE {
				fmt.Println("File is downloading")
			}
			return true
		} else if lookup.State == Failed {
			if DEBUGFILE {
				fmt.Println("Last time failed, but we will try this again")
			}
		}
	}
	// extract the information out
	g.fileHandler.searchFiles.Mux.Lock()
	_, present = g.fileHandler.searchFiles.searchedFiles[metafileHash]
	g.fileHandler.searchFiles.Mux.Unlock()

	if !present {
		fmt.Println("ERROR! The file has not been searched")
		return false
	}

	var result bool

	go func() {
		// build up the file
		file := &File{
			MetafileHash: metafileHash,
			State:        Downloading,
		}

		// put file into our file system
		g.fileHandler.filesLock.Lock()
		g.fileHandler.files[metafileHash] = file
		g.fileHandler.filesLock.Unlock()

		// get the metafile dest first
		// just choose the first one
		downloadList, valid := g.GenerateDownloadMapping(metafileHash)

		if !valid {
			fmt.Println("Cannot Generate Download List")
			result = false
			return
		}

		// start download metafile
		downloadReq := &DownloadRequest{
			Hash:     metafileHash,
			Dest:     downloadList[0],
			isMeta:   true,
			FileName: localFileName,
		}

		metaFileReply, valid := g.RequestFileChunk(downloadReq)

		if !valid {
			if DEBUGFILE {
				fmt.Println("We did not get the valid metaFile")
			}
			result = false
			return
		}

		metachunks, valid := g.fileHandler.addMeta(metaFileReply)
		g.fileHandler.filesLock.Lock()
		g.fileHandler.files[metafileHash].ChunkCount = uint64(len(metachunks))
		g.fileHandler.filesLock.Unlock()

		if !valid {
			if DEBUGSEARCH {
				fmt.Println("Cannot add Meta")
			}

			g.fileHandler.filesLock.Lock()
			g.fileHandler.files[metafileHash].State = Failed
			g.fileHandler.filesLock.Unlock()
			result = false
			return
		}

		originData := make([]byte, 0)
		chunkMap := make([]uint64, 0)

		for i, meta := range metachunks {
			// fmt.Printf("MetaChunk %d \n", i+1)
			// fmt.Println(hex.EncodeToString(meta[:]))

			// OUTPUT
			// fmt.Printf("DOWNLOADING %s chunk %d from %s \n", localFileName, i+1, dest)

			downloadReq = &DownloadRequest{
				Hash:     meta,
				Dest:     downloadList[i+1],
				isMeta:   false,
				SeqNum:   i + 1,
				FileName: localFileName,
			}

			localData, present := g.fileHandler.checkLocalChunk(meta)

			if present {
				if DEBUGSEARCH {
					fmt.Println("Find the chunk in local")
				}

				originData = append(originData, localData...)

				chunkMap = append(chunkMap, uint64(i+1))

				continue
			}

			dataReply, valid := g.RequestFileChunk(downloadReq)

			if !valid {
				if DEBUGFILE {
					fmt.Printf("Download Chunk %d Failed\n", i+1)
					fmt.Printf("Download %s terminated! \n", localFileName)
				}
				g.fileHandler.filesLock.Lock()
				g.fileHandler.files[metafileHash].State = Failed
				g.fileHandler.filesLock.Unlock()
				result = false
				return
			}

			originData = append(originData, dataReply.Data...)
			chunkMap = append(chunkMap, uint64(i+1))
		}
		g.fileHandler.combineChunks(metafileHash, originData, localFileName, chunkMap)
		result = true

		if DEBUGSEARCH {
			fmt.Println(g.fileHandler.files[metafileHash].ChunkMap)
			fmt.Println(g.fileHandler.files[metafileHash].ChunkCount)
		}

		// OUTPUT
		fmt.Printf("RECONSTRUCTED file %s \n", localFileName)
		return
	}()

	return result
}

func (g *Gossiper) GenerateDownloadMapping(metafileHash SHA256_HASH) ([]string, bool) {
	g.fileHandler.searchFiles.Mux.Lock()
	searchedInfo, present := g.fileHandler.searchFiles.searchedFiles[metafileHash]
	g.fileHandler.searchFiles.Mux.Unlock()

	if !present {
		fmt.Println("Searched File doesnot exit")
		return nil, false
	}

	chunkCount := searchedInfo.ChunkCount

	metaNeeded := true

	// init downloadList []string
	downloadList := make([]string, chunkCount+1)

	for origin, chunkMap := range searchedInfo.chunkSrc {
		if metaNeeded {
			metaNeeded = false
			downloadList[0] = origin
		}

		for _, ind := range chunkMap.ChunkMap {
			downloadList[ind] = origin
		}
	}

	// TODO: need to check the completion???

	return downloadList, true
}
