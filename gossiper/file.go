package gossiper

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
)

const SHARED_DIR = "_SharedFiles"

// const DOWNLOAD_DIR = "_SharedDir"

const DOWNLOAD_DIR = "_Downloads"
const CHUNK_SIZE = 8 * 1024

const DOWNLOAD_TIMEOUT = 5 * time.Second
const DOWNLOAD_RETRIES = 5

const NUM_DOWNLOAD_ROUTINE = 5

type SHA256_HASH [sha256.Size]byte

type FileType int

const (
	Shared      FileType = 0
	Downloaded  FileType = 1
	Downloading FileType = 2
	Failed      FileType = 3
)

type File struct {
	Name         string
	Size         int64
	Metafile     []byte
	MetafileHash SHA256_HASH
	State        FileType
}

type FileChunk struct {
	Data []byte
}

type DownloadRequest struct {
	Hash SHA256_HASH
	Dest string
}

type FileHandler struct {
	Name            string
	files           map[SHA256_HASH]*File
	filesLock       *sync.Mutex
	fileChunks      map[SHA256_HASH]*FileChunk
	fileChunksLock  *sync.Mutex
	sharedDir       string
	downloadDir     string
	requestTaskChan chan<- *DownloadRequest
	fileDispatcher  *FileDispatcher
}

func (g *Gossiper) HandleDownloadRequest(cmw *ClientMessageWrapper) {
	dest := *cmw.msg.Destination
	hash := *cmw.msg.Request

	shaHash, err := HashToSha256(hash)

	if err != nil {
		return
	}

	localFileName := *cmw.msg.File
	g.RequestFile(dest, shaHash, localFileName)
}

func (g *Gossiper) HandleFileIndexing(cmw *ClientMessageWrapper) {
	filename := *cmw.msg.File
	g.fileHandler.FileIndexingRequest(filename)
}

func tryCreateDir(abspath string) error {
	// refer to https://stackoverflow.com/questions/37932551/mkdir-if-not-exists-using-golang
	_, err := os.Stat(abspath)
	if os.IsNotExist(err) {
		// Dir is not exist then create Dir
		os.Mkdir(abspath, 0775)
	} else if err != nil {
		// This is other type of err
		return err
	}

	return nil
}

func NewFileHandler(name string) *FileHandler {
	exePath, err := os.Executable()

	if err != nil {
		panic(err)
	}

	sharedDir := filepath.Join(filepath.Dir(exePath), SHARED_DIR)
	if err = tryCreateDir(sharedDir); err != nil {
		panic(err)
	}

	downloadDir := filepath.Join(filepath.Dir(exePath), DOWNLOAD_DIR)
	if err = tryCreateDir(downloadDir); err != nil {
		panic(err)
	}

	return &FileHandler{
		Name:           name,
		files:          make(map[SHA256_HASH]*File),
		filesLock:      &sync.Mutex{},
		fileChunks:     make(map[SHA256_HASH]*FileChunk),
		fileChunksLock: &sync.Mutex{},
		sharedDir:      sharedDir,
		downloadDir:    downloadDir,
	}
}

func (g *Gossiper) RunFileSystem() {
	// Launch the file dispatcher
	g.fileHandler.fileDispatcher = LaunchFileDispatcher()

	// Initiallize the new requestchan not need till now
	// g.fileHandler.requestTaskChan = g.RunRequestChan()
}

// func (g *Gossiper) RunRequestChan() chan *DownloadRequest {
// 	reqChan := make(chan *DownloadRequest, CHANNEL_BUFFER_SIZE)

// 	for i := 0; i < NUM_DOWNLOAD_ROUTINE; i++ {
// 		go func() {
// 			for {
// 				select {
// 				case req, ok := <-reqChan:
// 					if !ok {
// 						// The channel has been closed here
// 						return
// 					}

// 					reply, success := g.RequestFileChunk(req)
// 					// donot need to further check the reply as it has already been checked in the RequestFileChunk

// 					if success {
// 						// download successful and accept the chunk
// 						g.fileHandler.acceptFileChunk(reply)
// 					} else {
// 						// TODO: Abort the download here
// 					}
// 				}
// 			}
// 		}()
// 	}

// 	return reqChan
// }

func (g *Gossiper) RequestFile(dest string, metafileHash SHA256_HASH, localFileName string) bool {
	// whether the file has already in local or not

	// Here, we will not consider whether it is in local
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

	var result bool

	go func() {

		// Generate file
		file := &File{
			MetafileHash: metafileHash,
			State:        Downloading,
		}

		// put file into our file system
		g.fileHandler.filesLock.Lock()
		g.fileHandler.files[metafileHash] = file
		g.fileHandler.filesLock.Unlock()

		downloadReq := &DownloadRequest{
			Hash: metafileHash,
			Dest: dest,
		}

		fmt.Printf("DOWNLOADING metafile of %s from %s \n", localFileName, dest)

		metaFileReply, valid := g.RequestFileChunk(downloadReq)

		if !valid {
			if DEBUGFILE {
				fmt.Println("We did not get the valid metaFile")
			}
			result = false
			return
		}

		metachunks, valid := g.fileHandler.addMeta(metaFileReply)

		if !valid {
			if DEBUGFILE {
				fmt.Println("Cannot add Meta")
			}

			g.fileHandler.filesLock.Lock()
			g.fileHandler.files[metafileHash].State = Failed
			g.fileHandler.filesLock.Unlock()
			result = false
			return
		}

		originData := make([]byte, 0)

		// send the dataRequest
		for i, meta := range metachunks {
			// fmt.Printf("MetaChunk %d \n", i+1)
			// fmt.Println(hex.EncodeToString(meta[:]))

			// OUTPUT
			fmt.Printf("DOWNLOADING %s chunk %d from %s \n", localFileName, i+1, dest)

			downloadReq = &DownloadRequest{
				Hash: meta,
				Dest: dest,
			}

			localData, present := g.fileHandler.checkLocalChunk(meta)

			if present {
				if DEBUGFILE {
					fmt.Println("Find the chunk in local")
				}

				originData = append(originData, localData...)
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
		}

		// Successful
		g.fileHandler.combineChunks(metafileHash, originData, localFileName)
		result = true

		// OUTPUT
		fmt.Printf("RECONSTRUCTED file %s \n", localFileName)
		return

	}()
	return result
}

func (g *Gossiper) RequestFileChunk(req *DownloadRequest) (*DataReply, bool) {
	dest := req.Dest
	shaHash := req.Hash

	hash, err := Sha256ToHash(shaHash)

	if err != nil {
		return nil, false
	}

	// SendRequest
	request := &DataRequest{
		Origin:      g.name,
		Destination: dest,
		HopLimit:    HOPLIMIT,
		HashValue:   hash,
	}

	success := g.RouteDataRequest(request)

	if !success {
		return nil, false
	}

	// put register the data
	replyListener := make(chan *DataReply, CHANNEL_BUFFER_SIZE)
	g.fileHandler.fileDispatcher.Register(shaHash, replyListener)

	ticker := time.NewTicker(DOWNLOAD_TIMEOUT)
	numRetries := 0

	defer ticker.Stop()
	defer g.fileHandler.fileDispatcher.Unregister(shaHash)

	for {
		select {
		case dataReply, ok := <-replyListener:
			if !ok {
				return nil, false
			}

			valid := g.fileHandler.checkReply(shaHash, dataReply)
			if !valid {
				if DEBUGFILE {
					fmt.Println("This is not valid reply")
				}

				// Based on the understanding last paragraph in the page 8 in Homework2 handout
				ticker = time.NewTicker(DOWNLOAD_TIMEOUT)
				continue
			}

			return dataReply, true

		case <-ticker.C:
			numRetries++

			if numRetries == DOWNLOAD_RETRIES {
				if DEBUGFILE {
					fmt.Println("Reach max retries")
				}

				return nil, false
			}
			// resent
			g.RouteDataRequest(request)
		}
	}

	return nil, false
}

func (g *Gossiper) HandleDataRequest(dataReq *DataRequest, sender *net.UDPAddr) {
	if DEBUGFILE {
		fmt.Printf("Receive DataRequest from %s to %s \n", dataReq.Origin, dataReq.Destination)
	}

	// check whether we have already have the result
	reply, present := g.fileHandler.checkFile(dataReq)

	if present {
		if DEBUGFILE {
			fmt.Println("Find DataReply")
		}
		tmp := sha256.Sum256(reply.Data)

		if DEBUGFILE {
			fmt.Println(hex.EncodeToString(tmp[:]))
		}

		g.RouteDataReply(reply)
		return
	}

	if g.name == dataReq.Destination {
		if DEBUGFILE {
			fmt.Printf("Destination Arrived! But %s does not exist the file needed \n", g.name)
		}

		emptyReply := &DataReply{
			Origin:      dataReq.Destination,
			Destination: dataReq.Origin,
			HopLimit:    HOPLIMIT,
			HashValue:   dataReq.HashValue,
			Data:        make([]byte, 0),
		}

		g.RouteDataReply(emptyReply)

		return
	}

	// check hoplimit
	request, valid := g.fileHandler.prepareNewRequest(dataReq)

	if !valid {
		return
	}

	routeSuccess := g.RouteDataRequest(request)

	if !routeSuccess {
		if DEBUGFILE {
			fmt.Println("Route Data Reply Fails")
		}
	}
}

func (g *Gossiper) HandleDataReply(dataReply *DataReply, sender *net.UDPAddr) {
	// fmt.Println(dataReply)
	if DEBUGFILE {
		fmt.Printf("Receive DataReply from %s to %s \n", dataReply.Origin, dataReply.Destination)
	}

	// check whether we have the file
	if g.name == dataReply.Destination {
		g.fileHandler.fileDispatcher.replyChan <- dataReply
		return
	}

	// check hoplimit
	reply, valid := g.fileHandler.prepareNewReply(dataReply)

	if !valid {
		return
	}

	success := g.RouteDataReply(reply)
	if !success {
		if DEBUGFILE {
			fmt.Println("Route Data Reply Fails")
		}
	}

}

func (f *FileHandler) acceptFileChunk(reply *DataReply) {
	// add the reply to the filechunks in the filehandler
	shaHash, err := HashToSha256(reply.HashValue)
	if err != nil {
		return
	}

	f.fileChunksLock.Lock()
	_, present := f.fileChunks[shaHash]
	f.fileChunksLock.Unlock()

	if present {
		return
	}

	f.fileChunksLock.Lock()
	f.fileChunks[shaHash] = &FileChunk{
		Data: reply.Data,
	}
	f.fileChunksLock.Unlock()
}

func (f *FileHandler) addMeta(reply *DataReply) ([]SHA256_HASH, bool) {
	// check the size of the metadata
	if len(reply.Data)%sha256.Size != 0 {
		if DEBUGFILE {
			fmt.Println("The size of metafile is not multiple of 32 bytes")
		}
		return nil, false
	}

	shaHash, err := HashToSha256(reply.HashValue)
	metafile := reply.Data

	if err != nil {
		return nil, false
	}

	f.filesLock.Lock()
	file, present := f.files[shaHash]
	f.filesLock.Unlock()

	if !present {
		if DEBUGFILE {
			fmt.Println("File Not Exist")
		}
		return nil, false
	} else if file.State != Downloading {
		if DEBUGFILE {
			fmt.Println("File Has Been Already In Local")
		}
		return nil, false
	} else {
		file.Metafile = metafile
	}

	numChunks := len(metafile) / sha256.Size

	if DEBUGFILE {
		fmt.Printf("The number of chunk is %d \n", numChunks)
	}

	splitedMeta := make([]SHA256_HASH, 0)

	for i := 0; i < numChunks; i++ {
		metaChunk, _ := HashToSha256(metafile[i*sha256.Size : (i+1)*sha256.Size])
		splitedMeta = append(splitedMeta, metaChunk)
	}

	return splitedMeta, true
}

func (f *FileHandler) checkLocalChunk(shaHash SHA256_HASH) ([]byte, bool) {
	f.filesLock.Lock()
	localfile, present := f.fileChunks[shaHash]
	f.filesLock.Unlock()

	if !present {
		return nil, false
	}
	return localfile.Data, true
}

func (f *FileHandler) checkFile(dataReq *DataRequest) (*DataReply, bool) {
	sha, err := HashToSha256(dataReq.HashValue)

	if err != nil {
		panic(err)
		return nil, false
	}

	// firstly check whether it is a metafile
	newReply := &DataReply{
		Origin:      dataReq.Destination,
		Destination: dataReq.Origin,
		HopLimit:    HOPLIMIT,
		HashValue:   dataReq.HashValue,
	}

	if metafile, present := f.files[sha]; present {
		if DEBUGFILE {
			fmt.Printf("MetaFile [%x] exist for Request from %s to %s \n", sha[:], dataReq.Origin, dataReq.Destination)
		}
		newReply.Data = metafile.Metafile
		return newReply, true
	}

	if file, present := f.fileChunks[sha]; present {
		if DEBUGFILE {
			fmt.Printf("Chunk [%x] exist for Request from %s to %s \n", sha[:], dataReq.Origin, dataReq.Destination)
		}
		newReply.Data = file.Data
		// fmt.Printf("The sha here is %s \n", hex.EncodeToString(sha[:]))
		// tmp := sha256.Sum256(newReply.Data)
		// fmt.Printf("The data here is %s \n", hex.EncodeToString(tmp[:]))
		return newReply, true
	}
	return nil, false
}

func (f *FileHandler) combineChunks(meshfileHash SHA256_HASH, data []byte, fileName string) {
	f.filesLock.Lock()
	f.files[meshfileHash].State = Downloaded
	f.files[meshfileHash].Name = fileName
	f.files[meshfileHash].Size = int64(len(data))
	f.filesLock.Unlock()
	localFile, err := os.OpenFile(filepath.Join(f.downloadDir, fileName), os.O_CREATE|os.O_WRONLY, 0755)

	if err != nil {
		panic(err)
	}

	localFile.Write(data)

	localFile.Close()

}

func (f *FileHandler) FileIndexingRequest(filename string) {
	abspath := filepath.Join(f.sharedDir, filename)

	fileIndexed, err := f.FileIndexing(abspath)

	if err != nil {
		fmt.Printf("%s does not exist \n", abspath)
		return
	}

	fileIndexed.Name = filename

	if DEBUGFILE {
		fmt.Println("File size is ", fileIndexed.Size)
	}

	f.filesLock.Lock()

	f.files[fileIndexed.MetafileHash] = fileIndexed

	f.filesLock.Unlock()

	metafileHash := fileIndexed.MetafileHash

	if DEBUGFILE {
		fmt.Println(hex.EncodeToString(metafileHash[:]))
	}
	// fmt.Println("FURTHER VERIFIED")
	// for h, k := range f.fileChunks {
	// 	fmt.Println("Key")
	// 	fmt.Println(hex.EncodeToString(h[:]))

	// 	fmt.Println("Value")
	// 	tmp := sha256.Sum256(k.Data)
	// 	fmt.Println(hex.EncodeToString(tmp[:]))
	// }
	// QishanWang metafileHash
	// 469403655c3a182a6b7856052a2428ebd24fede9e39b6cb428c21b8a0c222cc4
}

func (f *FileHandler) FileIndexing(abspath string) (*File, error) {
	// Refer to https://kgrz.io/reading-files-in-go-an-overview.html#reading-a-file-in-chunks
	// https://gobyexample.com/reading-files

	file, err := os.Open(abspath)

	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	defer file.Close()

	metaFile := make([]byte, 0)

	chunk := make([]byte, CHUNK_SIZE)

	f.fileChunksLock.Lock()

	for {
		bytesread, err := file.Read(chunk)

		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
				return nil, err
			}
			break
		}
		// Here is our one
		data := make([]byte, bytesread)
		copy(data, chunk)
		hash := sha256.Sum256(data)
		// convert [32]byte to []byte
		metaFile = append(metaFile, hash[:]...)

		// the chuck into the filehandler
		// check the hash exist or not

		_, present := f.fileChunks[hash]

		if present {
			if DEBUGFILE {
				fmt.Println("Wierd! The chunk has already existed!")
			}
		}

		newChunk := &FileChunk{
			Data: data,
		}
		// verify := sha256.Sum256(newChunk.Data)
		// fmt.Printf("Verified: %s \n", hex.EncodeToString(verify[:]))

		f.fileChunks[hash] = newChunk

	}

	f.fileChunksLock.Unlock()

	if DEBUGFILE {
		fmt.Printf("Size of metafile %d \n", len(metaFile))
	}

	metaFileHash := sha256.Sum256(metaFile)
	// fmt.Printf("Metafile Hash %s \n", hex.EncodeToString(metaFileHash[:]))

	fileStats, _ := file.Stat()

	fileIndexed := &File{
		Name:         abspath,
		Size:         fileStats.Size(),
		Metafile:     metaFile,
		MetafileHash: metaFileHash,
		State:        Shared,
	}

	return fileIndexed, nil
}

func (f *FileHandler) checkReply(originHash SHA256_HASH, dataReply *DataReply) bool {
	// check the hash is matched or not
	receivedHash, _ := HashToSha256(dataReply.HashValue)
	if receivedHash != originHash {
		if DEBUGFILE {
			fmt.Println("CHECK REPLY HASH DOES NOT MATCH")
		}
		return false
	}

	// what is more, we can also check whether data and hash match or not
	generatedHash := sha256.Sum256(dataReply.Data)
	// fmt.Println(hex.EncodeToString(generatedHash[:]))
	// fmt.Println(hex.EncodeToString(receivedHash[:]))
	if generatedHash != receivedHash {
		if DEBUGFILE {
			fmt.Println("GENERATE NEW HASH DOES NOT MATCH")
		}
		return false
	}

	return true
}

func Sha256ToHash(s SHA256_HASH) ([]byte, error) {
	var res = make([]byte, sha256.Size)

	if len(s) != sha256.Size {
		return res, errors.New("Size of sha256 is not 32 bytes")
	}

	copy(res, s[:])

	return res, nil
}

func HashToSha256(h []byte) (SHA256_HASH, error) {
	var res SHA256_HASH

	if len(h) != sha256.Size {
		if DEBUGFILE {
			return res, errors.New("Size of hash is not 32 bytes")
		}
	}
	copy(res[:], h)

	return res, nil
}

func (g *Gossiper) RouteDataRequest(dataReq *DataRequest) bool {
	dest := dataReq.Destination
	nextNode, present := g.routeTable.routeTable[dest]

	if !present {
		if DEBUGFILE {
			fmt.Println("Destination %s does not exist in the table \n", dest)
		}
		return false
	}

	if DEBUGFILE {
		fmt.Printf("Send the datarequest Dest: %s to Nextnode %s \n", dest, nextNode)
	}

	g.SendGossipPacketStrAddr(&GossipPacket{DataRequest: dataReq}, nextNode)

	return true
}

func (g *Gossiper) RouteDataReply(dataReply *DataReply) bool {
	dest := dataReply.Destination
	nextNode, present := g.routeTable.routeTable[dest]

	if !present {
		if DEBUGFILE {
			fmt.Println("Destination %s does not exist in the table \n", dest)
		}
		return false
	}

	if DEBUGFILE {
		fmt.Printf("Send the datareply Dest: %s to Nextnode %s \n", dest, nextNode)
	}

	g.SendGossipPacketStrAddr(&GossipPacket{DataReply: dataReply}, nextNode)

	return true
}

func (f *FileHandler) prepareNewRequest(dataReq *DataRequest) (*DataRequest, bool) {
	if dataReq.HopLimit == 0 {
		if DEBUGFILE {
			fmt.Println("HopLimit has been ended")
		}
		return nil, false
	}

	newDataRequest := &DataRequest{
		Origin:      dataReq.Origin,
		Destination: dataReq.Destination,
		HopLimit:    dataReq.HopLimit - 1,
		HashValue:   dataReq.HashValue,
	}

	return newDataRequest, true
}

func (f *FileHandler) prepareNewReply(dataReply *DataReply) (*DataReply, bool) {
	if dataReply.HopLimit == 0 {
		if DEBUGFILE {
			fmt.Println("HopLimit has been ended")
		}
		return nil, false
	}

	newDataReply := &DataReply{
		Origin:      dataReply.Origin,
		Destination: dataReply.Destination,
		HopLimit:    dataReply.HopLimit - 1,
		HashValue:   dataReply.HashValue,
		Data:        dataReply.Data,
	}
	return newDataReply, true
}
