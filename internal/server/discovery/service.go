package discovery

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/skenzeriq/patchiq/internal/shared/config"
)

// EventEmitter abstracts domain event emission for discovery.
type EventEmitter interface {
	EmitPatchDiscovered(ctx context.Context, tenantID, patchID, patchName, version, sourceRepo string) error
	EmitRepositorySynced(ctx context.Context, tenantID, repoName string, patchCount int) error
}

// Service orchestrates patch discovery: fetch -> parse -> upsert -> emit events.
type Service struct {
	upserter PatchUpserter
	emitter  EventEmitter
	fetcher  *Fetcher
}

// NewService creates a discovery Service.
func NewService(upserter PatchUpserter, emitter EventEmitter, fetcher *Fetcher) *Service {
	return &Service{upserter: upserter, emitter: emitter, fetcher: fetcher}
}

// DiscoverRepo fetches and parses a single repository, upserting patches.
func (s *Service) DiscoverRepo(ctx context.Context, tenantID string, repo config.RepositoryConfig) (int, error) {
	slog.InfoContext(ctx, "discovery: fetching repo", "repo", repo.Name, "url", repo.URL)
	body, err := s.fetcher.Fetch(ctx, repo.URL)
	if err != nil {
		return 0, fmt.Errorf("discover repo %s: %w", repo.Name, err)
	}
	defer body.Close()
	return s.discoverFromReader(ctx, tenantID, repo, body)
}

func (s *Service) discoverFromReader(ctx context.Context, tenantID string, repo config.RepositoryConfig, r io.Reader) (int, error) {
	parser, err := s.parserForRepo(repo)
	if err != nil {
		return 0, fmt.Errorf("discover repo %s: %w", repo.Name, err)
	}

	batch, err := s.upserter.BeginBatch(ctx, tenantID)
	if err != nil {
		return 0, fmt.Errorf("discover repo %s: begin batch: %w", repo.Name, err)
	}
	defer batch.Rollback(ctx)

	var count int
	for patch, parseErr := range parser.Parse(ctx, r) {
		if parseErr != nil {
			return count, fmt.Errorf("discover repo %s: parse: %w", repo.Name, parseErr)
		}
		if err := patch.Validate(); err != nil {
			slog.WarnContext(ctx, "discovery: skipping invalid patch", "error", err, "repo", repo.Name)
			continue
		}
		patchID, isNew, upsertErr := batch.UpsertPatch(ctx, patch)
		if upsertErr != nil {
			return count, fmt.Errorf("discover repo %s: upsert %s: %w", repo.Name, patch.Name, upsertErr)
		}
		count++
		if isNew {
			if err := s.emitter.EmitPatchDiscovered(ctx, tenantID, patchID, patch.Name, patch.Version, patch.SourceRepo); err != nil {
				slog.ErrorContext(ctx, "discovery: emit patch.discovered failed", "error", err, "patch", patch.Name)
			}
		}
	}

	if err := batch.Commit(ctx); err != nil {
		return count, fmt.Errorf("discover repo %s: commit batch: %w", repo.Name, err)
	}

	if err := s.emitter.EmitRepositorySynced(ctx, tenantID, repo.Name, count); err != nil {
		return count, fmt.Errorf("discover repo %s: emit repository.synced: %w", repo.Name, err)
	}
	slog.InfoContext(ctx, "discovery: repo sync complete", "repo", repo.Name, "patches", count)
	return count, nil
}

func (s *Service) parserForRepo(repo config.RepositoryConfig) (Parser, error) {
	switch repo.Type {
	case "apt":
		return &APTParser{OsFamily: repo.OsFamily, OsDistro: repo.OsDistro, SourceRepo: repo.URL}, nil
	case "yum":
		return &YUMParser{OsFamily: repo.OsFamily, OsDistro: repo.OsDistro, SourceRepo: repo.URL}, nil
	default:
		return nil, fmt.Errorf("unsupported repo type %q for repo %s", repo.Type, repo.Name)
	}
}
