package discovery

import (
	"context"
	"fmt"
	"testing"

	"github.com/skenzeriq/patchiq/internal/shared/config"
)

type mockBatchUpserter struct {
	patches   []DiscoveredPatch
	committed bool
	isNewFunc func(name string) bool
	upsertErr error
}

func (m *mockBatchUpserter) UpsertPatch(_ context.Context, p DiscoveredPatch) (string, bool, error) {
	if m.upsertErr != nil {
		return "", false, m.upsertErr
	}
	m.patches = append(m.patches, p)
	isNew := true
	if m.isNewFunc != nil {
		isNew = m.isNewFunc(p.Name)
	}
	return "patch-id-" + p.Name, isNew, nil
}

func (m *mockBatchUpserter) Commit(_ context.Context) error {
	m.committed = true
	return nil
}

func (m *mockBatchUpserter) Rollback(_ context.Context) {}

type mockUpserter struct {
	batches []*mockBatchUpserter
	current *mockBatchUpserter
}

func (m *mockUpserter) BeginBatch(_ context.Context, _ string) (BatchUpserter, error) {
	b := &mockBatchUpserter{}
	if m.current != nil {
		b.isNewFunc = m.current.isNewFunc
		b.upsertErr = m.current.upsertErr
	}
	m.batches = append(m.batches, b)
	m.current = b
	return b, nil
}

type mockEventEmitter struct {
	events  []string
	syncErr error
}

func (m *mockEventEmitter) EmitPatchDiscovered(_ context.Context, _, _, patchName, _, _ string) error {
	m.events = append(m.events, "patch.discovered:"+patchName)
	return nil
}

func (m *mockEventEmitter) EmitRepositorySynced(_ context.Context, _, repoName string, _ int) error {
	m.events = append(m.events, "repository.synced:"+repoName)
	return m.syncErr
}

func TestService_DiscoverRepo_APT(t *testing.T) {
	raw := "Package: curl\nVersion: 7.81.0\nArchitecture: amd64\nSHA256: abc123\n\nPackage: openssl\nVersion: 3.0.2\nArchitecture: amd64\nSHA256: def456\n"
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	svc := NewService(upserter, emitter, nil)

	repo := config.RepositoryConfig{
		Name:     "test-repo",
		Type:     "apt",
		OsFamily: "debian",
		OsDistro: "ubuntu-22.04",
	}

	count, err := svc.discoverFromReader(context.Background(), "tenant-1", repo, gzipString(t, raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 patches, got %d", count)
	}
	if len(upserter.batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(upserter.batches))
	}
	if len(upserter.current.patches) != 2 {
		t.Fatalf("expected 2 upserts, got %d", len(upserter.current.patches))
	}
	if !upserter.current.committed {
		t.Fatal("expected batch to be committed")
	}
	// 2 patch.discovered + 1 repository.synced = 3 events
	if len(emitter.events) != 3 {
		t.Fatalf("expected 3 events, got %d: %v", len(emitter.events), emitter.events)
	}
}

func TestService_DiscoverRepo_YUM(t *testing.T) {
	raw := `<?xml version="1.0" encoding="UTF-8"?>
<metadata xmlns="http://linux.duke.edu/metadata/common" packages="1">
  <package type="rpm">
    <name>openssl</name>
    <arch>x86_64</arch>
    <version epoch="1" ver="3.0.7" rel="27.el9"/>
    <checksum type="sha256" pkgid="YES">abc123</checksum>
    <summary>OpenSSL</summary>
    <description>OpenSSL toolkit</description>
    <size package="1234567" installed="2345678" archive="3456789"/>
    <location href="Packages/openssl-3.0.7-27.el9.x86_64.rpm"/>
  </package>
</metadata>`

	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	svc := NewService(upserter, emitter, nil)

	repo := config.RepositoryConfig{
		Name:     "rhel-test",
		Type:     "yum",
		OsFamily: "rhel",
		OsDistro: "rhel-9",
	}

	count, err := svc.discoverFromReader(context.Background(), "tenant-1", repo, gzipString(t, raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 patch, got %d", count)
	}
	// Verify epoch is included in version
	patch := upserter.current.patches[0]
	if patch.Version != "1:3.0.7-27.el9" {
		t.Errorf("expected version 1:3.0.7-27.el9, got %q", patch.Version)
	}
}

func TestService_DiscoverRepo_IsNewFalse(t *testing.T) {
	raw := "Package: curl\nVersion: 7.81.0\nArchitecture: amd64\nSHA256: abc123\n"
	upserter := &mockUpserter{
		current: &mockBatchUpserter{
			isNewFunc: func(_ string) bool { return false },
		},
	}
	emitter := &mockEventEmitter{}
	svc := NewService(upserter, emitter, nil)

	repo := config.RepositoryConfig{
		Name:     "test-repo",
		Type:     "apt",
		OsFamily: "debian",
		OsDistro: "ubuntu-22.04",
	}

	count, err := svc.discoverFromReader(context.Background(), "tenant-1", repo, gzipString(t, raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 patch, got %d", count)
	}
	// Only repository.synced, no patch.discovered since isNew=false
	if len(emitter.events) != 1 {
		t.Fatalf("expected 1 event (repository.synced only), got %d: %v", len(emitter.events), emitter.events)
	}
	if emitter.events[0] != "repository.synced:test-repo" {
		t.Errorf("expected repository.synced:test-repo, got %q", emitter.events[0])
	}
}

func TestService_ParserForRepo_UnknownType(t *testing.T) {
	svc := NewService(nil, nil, nil)
	repo := config.RepositoryConfig{
		Name: "unknown-repo",
		Type: "zypper",
	}
	_, err := svc.parserForRepo(repo)
	if err == nil {
		t.Fatal("expected error for unknown repo type")
	}
}

func TestService_RepositorySyncedError(t *testing.T) {
	raw := "Package: curl\nVersion: 7.81.0\nArchitecture: amd64\nSHA256: abc123\n"
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{syncErr: fmt.Errorf("bus down")}
	svc := NewService(upserter, emitter, nil)

	repo := config.RepositoryConfig{
		Name:     "test-repo",
		Type:     "apt",
		OsFamily: "debian",
		OsDistro: "ubuntu-22.04",
	}

	_, err := svc.discoverFromReader(context.Background(), "tenant-1", repo, gzipString(t, raw))
	if err == nil {
		t.Fatal("expected error when repository.synced emission fails")
	}
}

func TestService_DiscoverRepo_InvalidPatchSkipped(t *testing.T) {
	// Patch with empty name should be skipped
	raw := "Package: \nVersion: 1.0\nArchitecture: amd64\n\nPackage: curl\nVersion: 7.81.0\nArchitecture: amd64\nSHA256: abc123\n"
	upserter := &mockUpserter{}
	emitter := &mockEventEmitter{}
	svc := NewService(upserter, emitter, nil)

	repo := config.RepositoryConfig{
		Name:     "test-repo",
		Type:     "apt",
		OsFamily: "debian",
		OsDistro: "ubuntu-22.04",
	}

	count, err := svc.discoverFromReader(context.Background(), "tenant-1", repo, gzipString(t, raw))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 valid patch, got %d", count)
	}
}
