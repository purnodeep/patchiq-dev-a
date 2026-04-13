package comms

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"time"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// ConnectionState represents the agent's connection to the server.
type ConnectionState int

const (
	Disconnected ConnectionState = iota
	Connected
)

// ReconnectConfig controls exponential backoff behavior.
type ReconnectConfig struct {
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	JitterFactor float64
}

// DefaultReconnectConfig returns standard reconnection settings.
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		InitialDelay: 1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		JitterFactor: 0.2,
	}
}

// CalculateBackoff computes the delay for the given attempt number.
func CalculateBackoff(cfg ReconnectConfig, attempt int) time.Duration {
	delay := float64(cfg.InitialDelay) * math.Pow(cfg.Multiplier, float64(attempt))
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	if cfg.JitterFactor > 0 {
		jitter := delay * cfg.JitterFactor
		delay = delay - jitter + (rand.Float64() * 2 * jitter)
	}
	return time.Duration(delay)
}

// Client manages the gRPC connection to the Patch Manager server.
type Client struct {
	addr      string
	reconnCfg ReconnectConfig
	logger    *slog.Logger

	mu    sync.RWMutex
	conn  *grpc.ClientConn
	state ConnectionState
	agent pb.AgentServiceClient
}

// NewClient creates a new gRPC client for the given server address.
func NewClient(addr string, reconnCfg ReconnectConfig, logger *slog.Logger) *Client {
	return &Client{
		addr:      addr,
		reconnCfg: reconnCfg,
		logger:    logger,
		state:     Disconnected,
	}
}

// Connect establishes the gRPC connection.
func (c *Client) Connect(ctx context.Context) error {
	conn, err := grpc.NewClient(c.addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		return fmt.Errorf("grpc new client %s: %w", c.addr, err)
	}

	c.mu.Lock()
	if c.conn != nil {
		// Close the old connection to avoid leaking it.
		if closeErr := c.conn.Close(); closeErr != nil {
			c.logger.Debug("closing previous connection", "error", closeErr)
		}
	}
	c.conn = conn
	c.agent = pb.NewAgentServiceClient(conn)
	c.state = Connected
	c.mu.Unlock()

	c.logger.Info("connected to server", "addr", c.addr)
	return nil
}

// ConnectWithRetry creates the gRPC client with exponential backoff retry on
// failure. Note: grpc.NewClient is lazy — it does not establish a TCP connection
// immediately. Actual connectivity is verified later via heartbeats. This method
// retries only if NewClient itself returns an error (e.g. invalid address).
func (c *Client) ConnectWithRetry(ctx context.Context) error {
	for attempt := 0; ; attempt++ {
		err := c.Connect(ctx)
		if err == nil {
			return nil
		}
		delay := CalculateBackoff(c.reconnCfg, attempt)
		c.logger.Warn("client creation failed, retrying", "attempt", attempt, "delay", delay, "error", err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// AgentService returns the gRPC client.
func (c *Client) AgentService() pb.AgentServiceClient {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.agent
}

// State returns the current connection state.
func (c *Client) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.conn != nil {
		if c.conn.GetState() == connectivity.Ready {
			return Connected
		}
	}
	return c.state
}

// GRPCEnroller wraps pb.AgentServiceClient to implement Enroller.
type GRPCEnroller struct {
	client pb.AgentServiceClient
}

// NewGRPCEnroller creates a GRPCEnroller from the given AgentServiceClient.
func NewGRPCEnroller(client pb.AgentServiceClient) *GRPCEnroller {
	return &GRPCEnroller{client: client}
}

// Enroll delegates to the gRPC Enroll RPC.
func (e *GRPCEnroller) Enroll(ctx context.Context, req *pb.EnrollRequest) (*pb.EnrollResponse, error) {
	return e.client.Enroll(ctx, req)
}

// GRPCHeartbeatStreamer wraps pb.AgentServiceClient to implement HeartbeatStreamer.
type GRPCHeartbeatStreamer struct {
	client pb.AgentServiceClient
}

// NewGRPCHeartbeatStreamer creates a GRPCHeartbeatStreamer from the given AgentServiceClient.
func NewGRPCHeartbeatStreamer(client pb.AgentServiceClient) *GRPCHeartbeatStreamer {
	return &GRPCHeartbeatStreamer{client: client}
}

// OpenHeartbeat opens a bidirectional heartbeat stream.
func (s *GRPCHeartbeatStreamer) OpenHeartbeat(ctx context.Context) (HeartbeatStream, error) {
	stream, err := s.client.Heartbeat(ctx)
	if err != nil {
		return nil, fmt.Errorf("open grpc heartbeat stream: %w", err)
	}
	return &grpcHeartbeatStream{stream: stream}, nil
}

// grpcHeartbeatStream wraps a gRPC bidi stream to implement HeartbeatStream.
type grpcHeartbeatStream struct {
	stream grpc.BidiStreamingClient[pb.HeartbeatRequest, pb.HeartbeatResponse]
}

func (s *grpcHeartbeatStream) Send(req *pb.HeartbeatRequest) error {
	return s.stream.Send(req)
}

func (s *grpcHeartbeatStream) Recv() (*pb.HeartbeatResponse, error) {
	return s.stream.Recv()
}

func (s *grpcHeartbeatStream) CloseSend() error {
	return s.stream.CloseSend()
}

// GRPCOutboxStreamer wraps pb.AgentServiceClient to implement OutboxStreamOpener.
type GRPCOutboxStreamer struct {
	client  pb.AgentServiceClient
	agentID string
}

// NewGRPCOutboxStreamer creates a GRPCOutboxStreamer from the given AgentServiceClient.
func NewGRPCOutboxStreamer(client pb.AgentServiceClient, agentID string) *GRPCOutboxStreamer {
	return &GRPCOutboxStreamer{client: client, agentID: agentID}
}

// OpenOutboxStream opens a bidirectional SyncOutbox stream with x-agent-id metadata.
func (s *GRPCOutboxStreamer) OpenOutboxStream(ctx context.Context) (OutboxSyncer, error) {
	md := metadata.Pairs("x-agent-id", s.agentID)
	ctx = metadata.NewOutgoingContext(ctx, md)
	stream, err := s.client.SyncOutbox(ctx)
	if err != nil {
		return nil, fmt.Errorf("open grpc outbox stream: %w", err)
	}
	return stream, nil
}

// RunConfig holds parameters for the agent connection lifecycle.
type RunConfig struct {
	DataDir         string
	Token           string
	Meta            AgentMeta
	Endpoint        *pb.EndpointInfo
	HeartbeatConfig HeartbeatConfig
	Outbox          *Outbox
	Inbox           *Inbox
	SyncConfig      SyncConfig
	// OnCommandsPending is called after the inbox is fetched when the server
	// signals pending commands. Use it to trigger additional processing
	// (e.g. the command processor loop). Optional.
	OnCommandsPending func()
}

// Run executes the full agent lifecycle: cert → connect → enroll → heartbeat loop.
func (c *Client) Run(ctx context.Context, state *AgentState, runCfg RunConfig) error {
	// Step 1: Load or generate TLS certificate (not used for transport in M1).
	if _, err := LoadOrGenerateCert(runCfg.DataDir, c.logger); err != nil {
		return fmt.Errorf("run load cert: %w", err)
	}

	// Step 2: Connect with retry.
	if err := c.ConnectWithRetry(ctx); err != nil {
		return fmt.Errorf("run connect: %w", err)
	}

	// Outer loop: enroll → heartbeat. Re-entered on re-enrollment requests.
	for {
		// Step 3: Enroll with retry (server may not be reachable yet).
		var result EnrollResult
		for enrollAttempt := 0; ; enrollAttempt++ {
			enroller := NewGRPCEnroller(c.AgentService())
			var enrollErr error
			result, enrollErr = Enroll(ctx, enroller, state, runCfg.Token, runCfg.Meta, runCfg.Endpoint)
			if enrollErr == nil {
				break
			}
			delay := CalculateBackoff(c.reconnCfg, enrollAttempt)
			c.logger.Warn("enrollment failed, retrying", "attempt", enrollAttempt, "delay", delay, "error", enrollErr)
			select {
			case <-ctx.Done():
				return fmt.Errorf("run enroll: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		// Step 4: Apply server config to heartbeat interval if returned.
		hbCfg := runCfg.HeartbeatConfig
		hbCfg.AgentID = result.AgentID
		hbCfg.ProtocolVersion = result.NegotiatedVersion
		if result.Config != nil && result.Config.HeartbeatIntervalSeconds > 0 {
			hbCfg.Interval = time.Duration(result.Config.HeartbeatIntervalSeconds) * time.Second
			c.logger.Info("using server heartbeat interval", "interval", hbCfg.Interval)
		}

		// Step 5: Heartbeat loop with reconnection + outbox sync.
		streamer := NewGRPCHeartbeatStreamer(c.AgentService())
		for attempt := 0; ; attempt++ {
			// Create a SyncRunner for this connection session.
			outboxStreamer := NewGRPCOutboxStreamer(c.AgentService(), result.AgentID)
			syncRunner := NewSyncRunnerWithOpener(outboxStreamer, runCfg.Outbox, runCfg.SyncConfig, c.logger)

			// Wire heartbeat's OnCommandsPending to trigger an immediate outbox sync,
			// inbox fetch (to receive pending commands from the server), and command
			// processing.
			inboxFetcher := NewGRPCInboxFetcher(c.AgentService(), result.AgentID)
			hbCfg.OnCommandsPending = func() {
				syncRunner.Trigger()
				if runCfg.Inbox != nil {
					go func() {
						if err := FetchInbox(ctx, inboxFetcher, runCfg.Inbox, c.logger); err != nil {
							c.logger.Warn("inbox fetch failed", "error", err)
						}
						// After fetching, trigger the command processor so it picks
						// up the newly stored commands without waiting for its ticker.
						if runCfg.OnCommandsPending != nil {
							runCfg.OnCommandsPending()
						}
					}()
				}
			}

			// Run sync runner in a child context so it stops when heartbeat fails.
			syncCtx, syncCancel := context.WithCancel(ctx)
			go func() {
				if err := syncRunner.Run(syncCtx); err != nil && syncCtx.Err() == nil {
					c.logger.Warn("sync runner exited with error", "error", err)
				}
			}()

			hbErr := RunHeartbeat(ctx, streamer, hbCfg, runCfg.Outbox)
			syncCancel() // stop sync runner when heartbeat ends

			if ctx.Err() != nil {
				return ctx.Err()
			}
			if errors.Is(hbErr, ErrReEnrollRequired) {
				c.logger.Info("server requested re-enrollment, clearing agent_id")
				if clearErr := state.Set(ctx, "agent_id", ""); clearErr != nil {
					return fmt.Errorf("clear agent_id for re-enrollment: %w", clearErr)
				}
				break // re-enter outer loop to re-enroll
			}
			if errors.Is(hbErr, ErrShutdownRequested) {
				return hbErr
			}

			delay := CalculateBackoff(c.reconnCfg, attempt)
			c.logger.Warn("heartbeat stream failed, reconnecting", "attempt", attempt, "delay", delay, "error", hbErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}

			// Re-establish connection and streamer for retry.
			if connErr := c.ConnectWithRetry(ctx); connErr != nil {
				return fmt.Errorf("run reconnect: %w", connErr)
			}
			streamer = NewGRPCHeartbeatStreamer(c.AgentService())
		}

		// Reconnect before re-enrolling.
		if connErr := c.ConnectWithRetry(ctx); connErr != nil {
			return fmt.Errorf("run reconnect for re-enrollment: %w", connErr)
		}
	}
}

// Close closes the gRPC connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		c.state = Disconnected
		return c.conn.Close()
	}
	return nil
}
