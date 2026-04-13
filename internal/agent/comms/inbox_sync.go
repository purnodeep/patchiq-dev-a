package comms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"google.golang.org/grpc/metadata"
)

// InboxFetcher abstracts the SyncInbox server-streaming RPC.
type InboxFetcher interface {
	FetchCommands(ctx context.Context) (InboxStream, error)
}

// InboxStream abstracts a server-streaming response of CommandRequest messages.
type InboxStream interface {
	Recv() (*pb.CommandRequest, error)
}

// GRPCInboxFetcher calls the SyncInbox RPC via gRPC.
type GRPCInboxFetcher struct {
	client  pb.AgentServiceClient
	agentID string
}

// NewGRPCInboxFetcher creates a GRPCInboxFetcher.
func NewGRPCInboxFetcher(client pb.AgentServiceClient, agentID string) *GRPCInboxFetcher {
	return &GRPCInboxFetcher{client: client, agentID: agentID}
}

// FetchCommands opens a SyncInbox stream and returns it.
func (f *GRPCInboxFetcher) FetchCommands(ctx context.Context) (InboxStream, error) {
	md := metadata.Pairs("x-agent-id", f.agentID)
	ctx = metadata.NewOutgoingContext(ctx, md)

	stream, err := f.client.SyncInbox(ctx, &pb.InboxRequest{
		AgentId:         f.agentID,
		ProtocolVersion: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("open inbox stream: %w", err)
	}
	return stream, nil
}

// FetchInbox calls SyncInbox and stores received commands in the local inbox.
// It reads all commands from the stream until EOF. Each command is stored
// idempotently (inbox.Store uses INSERT OR IGNORE on ID).
func FetchInbox(ctx context.Context, fetcher InboxFetcher, inbox *Inbox, logger *slog.Logger) error {
	stream, err := fetcher.FetchCommands(ctx)
	if err != nil {
		return fmt.Errorf("fetch inbox: %w", err)
	}

	count := 0
	storeErrors := 0
	for {
		cmd, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("fetch inbox recv: %w", err)
		}

		item := InboxItem{
			ID:          cmd.CommandId,
			CommandType: cmd.Type.String(),
			Payload:     cmd.Payload,
			Priority:    int(cmd.Priority),
			ReceivedAt:  time.Now().UTC().Format(time.RFC3339Nano),
			Status:      "pending",
		}

		if err := inbox.Store(ctx, item); err != nil {
			logger.WarnContext(ctx, "fetch inbox: store command failed", "command_id", cmd.CommandId, "error", err)
			storeErrors++
			continue
		}
		count++
	}

	if storeErrors > 0 && count == 0 {
		return fmt.Errorf("fetch inbox: all %d received commands failed to store", storeErrors)
	}
	if count > 0 {
		logger.InfoContext(ctx, "fetch inbox: received commands", "count", count)
	}
	return nil
}
