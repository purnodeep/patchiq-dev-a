//go:build windows

package patcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockWUAClient struct {
	searchResults []wuaUpdate
	searchErr     error
	downloadErr   error
	installErr    error
	installResult wuaInstallResult
}

func (m *mockWUAClient) SearchUpdates(_ context.Context, _ string) ([]wuaUpdate, error) {
	return m.searchResults, m.searchErr
}

func (m *mockWUAClient) DownloadUpdates(_ context.Context, _ []wuaUpdate) error {
	return m.downloadErr
}

func (m *mockWUAClient) InstallUpdates(_ context.Context, _ []wuaUpdate) (wuaInstallResult, error) {
	return m.installResult, m.installErr
}

// stubPreflightChecks overrides the admin and WUA service checks so unit
// tests that exercise Install logic are not blocked by real system state.
// It returns a cleanup function that restores the originals.
func stubPreflightChecks() func() {
	origAdmin := checkAdmin
	origWUA := checkWUAService
	checkAdmin = func() error { return nil }
	checkWUAService = func() error { return nil }
	return func() {
		checkAdmin = origAdmin
		checkWUAService = origWUA
	}
}

func TestWUAInstaller_Name(t *testing.T) {
	inst := &wuaInstaller{}
	assert.Equal(t, "wua", inst.Name())
}

func TestWUAInstaller_Install_Success(t *testing.T) {
	defer stubPreflightChecks()()

	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765 Cumulative", UpdateID: "abc-123"}},
			installResult: wuaInstallResult{ResultCode: 2, RebootRequired: false},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, false)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
	assert.False(t, result.RebootRequired)
}

func TestWUAInstaller_Install_RebootRequired(t *testing.T) {
	defer stubPreflightChecks()()

	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765", UpdateID: "abc-123"}},
			installResult: wuaInstallResult{ResultCode: 2, RebootRequired: true},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, false)
	require.NoError(t, err)
	assert.True(t, result.RebootRequired)
}

func TestWUAInstaller_Install_NotFound(t *testing.T) {
	defer stubPreflightChecks()()

	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{},
		},
	}

	_, err := inst.Install(context.Background(), PatchTarget{Name: "KB9999999"}, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no matching update found")
}

func TestWUAInstaller_Install_DryRun(t *testing.T) {
	defer stubPreflightChecks()()

	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765", UpdateID: "abc-123"}},
			installErr:    fmt.Errorf("should not be called"),
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, true)
	require.NoError(t, err)
	assert.Equal(t, 0, result.ExitCode)
}

func TestWUAInstaller_Install_Failed(t *testing.T) {
	defer stubPreflightChecks()()

	inst := &wuaInstaller{
		client: &mockWUAClient{
			searchResults: []wuaUpdate{{Title: "KB5034765", UpdateID: "abc-123"}},
			installResult: wuaInstallResult{ResultCode: 4, RebootRequired: false},
		},
	}

	result, err := inst.Install(context.Background(), PatchTarget{Name: "KB5034765"}, false)
	require.NoError(t, err)
	assert.Equal(t, 4, result.ExitCode)
}

func TestWUAInstaller_Install_AdminCheckFails(t *testing.T) {
	origCheck := checkAdmin
	checkAdmin = func() error { return errNotAdmin }
	defer func() { checkAdmin = origCheck }()

	w := &wuaInstaller{
		client: &mockWUAClient{},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
	_, err := w.Install(context.Background(), PatchTarget{Name: "KB123"}, false)
	if err == nil {
		t.Fatal("expected error when admin check fails")
	}
	if !errors.Is(err, errNotAdmin) {
		t.Errorf("error = %v, want wrapping errNotAdmin", err)
	}
}

func TestWUAInstaller_Install_WUAServiceStopped(t *testing.T) {
	origAdmin := checkAdmin
	origWUA := checkWUAService
	checkAdmin = func() error { return nil }
	checkWUAService = func() error { return errWUAServiceStopped }
	defer func() {
		checkAdmin = origAdmin
		checkWUAService = origWUA
	}()

	w := &wuaInstaller{
		client: &mockWUAClient{},
		logger: slog.New(slog.NewTextHandler(os.Stderr, nil)),
	}
	_, err := w.Install(context.Background(), PatchTarget{Name: "KB123"}, false)
	if err == nil {
		t.Fatal("expected error when WUA service is stopped")
	}
	if !errors.Is(err, errWUAServiceStopped) {
		t.Errorf("error = %v, want wrapping errWUAServiceStopped", err)
	}
}

func TestClassifyCOMError_AccessDenied(t *testing.T) {
	err := classifyCOMError(fmt.Errorf("COM error: 80070005"))
	if !errors.Is(err, errNotAdmin) {
		t.Errorf("expected errNotAdmin, got: %v", err)
	}
}

func TestClassifyCOMError_InstallNotAllowed(t *testing.T) {
	err := classifyCOMError(fmt.Errorf("COM error: 80240016"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Errorf("expected 'not allowed' in error, got: %v", err)
	}
}

func TestClassifyCOMError_Nil(t *testing.T) {
	if classifyCOMError(nil) != nil {
		t.Error("expected nil for nil input")
	}
}
