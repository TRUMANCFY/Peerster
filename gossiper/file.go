package gossiper

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	. "github.com/TRUMANCFY/Peerster/message"
)

const SHARED_DIR = "_SharedDir"
const DOWNLOAD_DIR = "_Downloads"
const CHUNK_SIZE = 8 * 1024

const DOWNLOAD_TIMEOUT = 5 * time.Second
const DOWNLOAD_RETRIEs = 5

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

type FileChuck struct {
	Data []byte
}

type DownloadRequest struct {
	Hash SHA256_HASH
	Dest string
}

type FileHandler struct {
	Name            string
	files           map[SHA256_HASH]*File
	fileChucks      map[SHA256_HASH]*FileChuck
	sharedDir       string
	downloadDir     string
	requestTaskChan chan<- *DownloadRequest
	fileDispatcher  *FileDispatcher
}

func tryCreateDir(abspath string) error {
	// refer to https://stackoverflow.com/questions/37932551/mkdir-if-not-exists-using-golang
	if _, err := os.Stat(abspath); os.IsNotExist(err) {
		// Dir is not exist then create Dir
		os.Mkdir(abspath, 0775)

		if err != nil {
			return err
		}
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
		Name:        name,
		files:       make(map[SHA256_HASH]*File),
		fileChucks:  make(map[SHA256_HASH]*FileChuck),
		sharedDir:   sharedDir,
		downloadDir: downloadDir,
	}
}

func (g *Gossiper) RunFileSystem() {
	// Launch the file dispatcher
	g.fileHandler.fileDispatcher = LaunchFileDispatcher()

	// Initiallize the new
	g.fileHandler.requestTaskChan = g.RunRequestChan()
}

func (g *Gossiper) RunRequestChan() chan *DownloadRequest {
	reqChan := make(chan *DownloadRequest, CHANNEL_BUFFER_SIZE)

	for i := 0; i < NUM_DOWNLOAD_ROUTINE; i++ {
		go func() {
			for {
				select {
				case req, ok := <-reqChan:
					if !ok {
						// TODO: Stop here?
						return
					}

					reply, success := g.RequestFileChunk(req)
					// TODO check with reply
				}
			}
		}()
	}

	return reqChan
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
				return nil, false
			}

			return dataReply, true

		case <-ticker.C:
			numRetries++

			if numRetries == DOWNLOAD_RETRIEs {
				fmt.Println("Reach max retries")
				return nil, false
			}
			// resent
			g.RouteDataRequest(request)
		}
	}

	return nil, false
}

func (g *Gossiper) HandleDataRequest(dataReq *DataRequest, sender *net.UDPAddr) {
	// check whether we have already have the result
	reply, present := g.fileHandler.checkFile(dataReq)

	if present {
		g.RouteDataReply(reply)
		return
	}

	if g.name == dataReq.Destination {
		fmt.Printf("Destination Arrived! But %s does not exist the file needed \n", g.name)
		return
	}

	// check hoplimit
	request, valid := g.fileHandler.prepareNewRequest(dataReq)

	if !valid {
		return
	}

	routeSuccess := g.RouteDataRequest(request)

	if !routeSuccess {
		fmt.Println("Route Data Reply Fails")
	}
}

func (g *Gossiper) HandleDataReply(dataReply *DataReply, sender *net.UDPAddr) {
	// check whether we have the file
	if g.name == dataReply.Destination {

	}

	// check hoplimit

	reply, valid := g.fileHandler.prepareNewReply(dataReply)

	if !valid {
		return
	}

	success := g.RouteDataReply(reply)
	if !success {
		fmt.Println("Route Data Reply Fails")
	}

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
		newReply.Data = metafile.Metafile
		return newReply, true
	}

	if file, present := f.fileChucks[sha]; present {
		newReply.Data = file.Data
		return newReply, true
	}
	return nil, false
}

func (f *FileHandler) FileIndexingRequest(filename string) {
	abspath := filepath.Join(f.sharedDir, filename)

	fileIndexed, err := f.FileIndexing(abspath)

	if err != nil {
		fmt.Printf("%s does not exist \n", abspath)
		return
	}

	fileIndexed.Name = filename

	f.files[fileIndexed.MetafileHash] = fileIndexed

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

	for {
		bytesread, err := file.Read(chunk)

		if err != nil {
			if err != io.EOF {
				fmt.Println(err)
				return nil, err
			}
			break
		}
		hash := sha256.Sum256(chunk[:bytesread])
		// convert [32]byte to []byte
		metaFile = append(metaFile, hash[:]...)

		// the chuck into the filehandler
		// check the hash exist or not
		_, present := f.fileChucks[hash]

		if present {
			fmt.Println("Wierd! The chunk has already existed!")
		}

		newChunk := &FileChuck{
			Data: chunk[:bytesread],
		}

		f.fileChucks[hash] = newChunk

	}

	metaFileHash := sha256.Sum256(metaFile)

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
		return false
	}

	// what is more, we can also check whether data and hash match or not
	generatedHash := sha256.Sum256(dataReply.Data)
	if generatedHash != receivedHash {
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
		return res, errors.New("Size of hash is not 32 bytes")
	}
	copy(res[:], h)

	return res, nil
}

func (g *Gossiper) RouteDataRequest(dataReq *DataRequest) bool {
	dest := dataReq.Destination
	nextNode, present := g.routeTable.routeTable[dest]

	if !present {
		fmt.Println("Destination %s does not exist in the table \n", dest)
		return false
	}

	g.SendGossipPacketStrAddr(&GossipPacket{DataRequest: dataReq}, nextNode)

	return true
}

func (g *Gossiper) RouteDataReply(dataReply *DataReply) bool {
	dest := dataReply.Destination
	nextNode, present := g.routeTable.routeTable[dest]

	if !present {
		fmt.Println("Destination %s does not exist in the table \n", dest)
		return false
	}

	g.SendGossipPacketStrAddr(&GossipPacket{DataReply: dataReply}, nextNode)

	return true
}

func (f *FileHandler) prepareNewRequest(dataReq *DataRequest) (*DataRequest, bool) {
	if dataReq.HopLimit == 0 {
		fmt.Println("HopLimit has been ended")
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
		fmt.Println("HopLimit has been ended")
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
