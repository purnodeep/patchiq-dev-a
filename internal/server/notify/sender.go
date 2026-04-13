package notify

import (
	"context"
	"fmt"

	"github.com/containrrr/shoutrrr"
)

// Sender sends a notification message to a Shoutrrr URL.
type Sender interface {
	Send(ctx context.Context, shoutrrrURL string, message string) error
}

// ShoutrrrSender implements Sender using the Shoutrrr library.
type ShoutrrrSender struct{}

func (s *ShoutrrrSender) Send(_ context.Context, shoutrrrURL string, message string) error {
	if err := shoutrrr.Send(shoutrrrURL, message); err != nil {
		return fmt.Errorf("shoutrrr send: %w", err)
	}
	return nil
}

// MockSender is a test double for Sender that records calls.
type MockSender struct {
	Err      error
	Calls    []string // Shoutrrr URLs that were sent to
	Messages []string // Messages that were sent
}

func (m *MockSender) Send(_ context.Context, shoutrrrURL string, message string) error {
	m.Calls = append(m.Calls, shoutrrrURL)
	m.Messages = append(m.Messages, message)
	return m.Err
}
