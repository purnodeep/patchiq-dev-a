package grpc

import (
	"log/slog"

	pb "github.com/skenzeriq/patchiq/gen/patchiq/v1"
	"github.com/skenzeriq/patchiq/internal/server/cve"
	"github.com/skenzeriq/patchiq/internal/server/store"
	"github.com/skenzeriq/patchiq/internal/shared/domain"
)

const (
	// ServerProtocolVersion is the current protocol version the server supports.
	ServerProtocolVersion uint32 = 1
	// ServerMinProtocolVersion is the minimum protocol version the server accepts.
	ServerMinProtocolVersion uint32 = 1
)

// AgentServiceServer implements the gRPC AgentService.
type AgentServiceServer struct {
	pb.UnimplementedAgentServiceServer
	store          *store.Store
	eventBus       domain.EventBus
	logger         *slog.Logger
	cveJobInserter cve.JobInserter
}

// SetCVEJobInserter configures the CVE match job inserter so that inventory
// receipt automatically triggers vulnerability correlation for the endpoint.
func (s *AgentServiceServer) SetCVEJobInserter(inserter cve.JobInserter) {
	s.cveJobInserter = inserter
}

// NewAgentServiceServer creates a new AgentServiceServer.
func NewAgentServiceServer(st *store.Store, eventBus domain.EventBus, logger *slog.Logger) *AgentServiceServer {
	if st == nil {
		panic("grpc: NewAgentServiceServer called with nil store")
	}
	if eventBus == nil {
		panic("grpc: NewAgentServiceServer called with nil eventBus")
	}
	if logger == nil {
		panic("grpc: NewAgentServiceServer called with nil logger")
	}
	return &AgentServiceServer{
		store:    st,
		eventBus: eventBus,
		logger:   logger,
	}
}
