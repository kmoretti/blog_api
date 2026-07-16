package bot

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const maxBotAttachmentBytes = int64(64 << 20)

var attachmentHTTPClient = &http.Client{Timeout: 2 * time.Minute}

func downloadToTemp(ctx context.Context, rawURL string) (*os.File, int64, []byte, error) {
	return downloadToTempLimit(ctx, rawURL, maxBotAttachmentBytes)
}

func downloadToTempLimit(ctx context.Context, rawURL string, maxBytes int64) (*os.File, int64, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create attachment request: %w", err)
	}
	resp, err := attachmentHTTPClient.Do(req)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("download attachment: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, nil, fmt.Errorf("download attachment: status %s", resp.Status)
	}
	if resp.ContentLength > maxBytes {
		return nil, 0, nil, fmt.Errorf("attachment exceeds %d bytes", maxBytes)
	}

	file, err := os.CreateTemp("", "blog-api-media-*")
	if err != nil {
		return nil, 0, nil, fmt.Errorf("create attachment temp file: %w", err)
	}
	keep := false
	defer func() {
		if !keep {
			file.Close()
			os.Remove(file.Name())
		}
	}()

	size, err := io.Copy(file, io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, 0, nil, fmt.Errorf("write attachment temp file: %w", err)
	}
	if size > maxBytes {
		return nil, 0, nil, fmt.Errorf("attachment exceeds %d bytes", maxBytes)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, 0, nil, fmt.Errorf("rewind attachment: %w", err)
	}

	sample := make([]byte, min(size, 512))
	if _, err := io.ReadFull(file, sample); err != nil {
		return nil, 0, nil, fmt.Errorf("read attachment sample: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, 0, nil, fmt.Errorf("rewind attachment: %w", err)
	}
	keep = true
	return file, size, sample, nil
}
