package loader

import (
	"io"
	"naviserver/internal/domain"
)

type ProgressReader struct {
	Reader       io.Reader
	Total        int64
	Current      int64
	ProgressChan chan<- domain.ProgressEvent
	ServerID     string
	Message      string
}

func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Current += int64(n)

	if pr.ProgressChan != nil && pr.Total > 0 {
		percentage := float64(pr.Current) / float64(pr.Total) * 100
		pr.ProgressChan <- domain.ProgressEvent{
			ServerID:     pr.ServerID,
			Message:      pr.Message,
			Progress:     percentage,
			CurrentBytes: pr.Current,
			TotalBytes:   pr.Total,
		}
	}

	return n, err
}
