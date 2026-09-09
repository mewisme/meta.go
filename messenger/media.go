package messenger

import (
	"context"
	"io"

	"go.mewis.me/fbgo/internal/webapi"
)

type MediaDownload struct {
	Body          io.ReadCloser
	ContentType   string
	ContentLength int64
}

func DownloadMedia(ctx context.Context, rawURL string, maxBytes int64) (*MediaDownload, error) {
	result, err := webapi.DownloadMedia(ctx, rawURL, maxBytes)
	if err != nil {
		return nil, err
	}
	return &MediaDownload{Body: result.Body, ContentType: result.ContentType, ContentLength: result.ContentLength}, nil
}
