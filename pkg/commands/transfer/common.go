package transfer

import (
	"crypto/sha256"
	"io"
	"os"
	"sync"
)

// The stream for downloading/uploading files from/to the client
const downloadStream = "downloadStream"

// Small shared buffer pool
var smallBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 1024)
	},
}

func getFileHash(f *os.File) ([]byte, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}

	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}
