package gossiper

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const SHARED_DIR = "_SharedDir"
const CHUNK_SIZE = 8 * 1024

type SHA256_HASH [sha256.Size]byte

type File struct {
	Name         string
	Size         int64
	Metafile     []byte
	MetafileHash SHA256_HASH
}

type Files struct {
	files     map[SHA256_HASH]*File
	sharedDir string
}

func NewFiles() *Files {
	return &Files{
		files:     make(map[SHA256_HASH]*File),
		sharedDir: SHARED_DIR,
	}
}

func (f *Files) FileIndexingRequest(filename string) {
	abspath := filepath.Join(f.sharedDir, filename)

	fileIndexed, err := f.FileIndexing(abspath)

	if err != nil {
		fmt.Printf("%s does not exist \n", abspath)
		return
	}

	fileIndexed.Name = filename

	f.files[fileIndexed.MetafileHash] = fileIndexed

}

func (f *Files) FileIndexing(abspath string) (*File, error) {
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
	}

	metaFileHash := sha256.Sum256(metaFile)

	fileStats, _ := file.Stat()

	fileIndexed := &File{
		Name:         abspath,
		Size:         fileStats.Size(),
		Metafile:     metaFile,
		MetafileHash: metaFileHash,
	}

	return fileIndexed, nil
}
