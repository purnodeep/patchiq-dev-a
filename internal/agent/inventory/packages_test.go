package inventory

import (
	"os/exec"
	"testing"
)

// noSnap returns a snapLookPath that always reports snap as unavailable.
func noSnap() func() (string, error) {
	return func() (string, error) { return "", exec.ErrNotFound }
}

func TestDetectCollectors_APTAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/dpkg_status_basic",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "linux",
	}

	collectors := detectCollectors(deps)

	if len(collectors) != 1 {
		t.Fatalf("expected 1 collector, got %d", len(collectors))
	}
	if collectors[0].Name() != "apt" {
		t.Errorf("expected collector name %q, got %q", "apt", collectors[0].Name())
	}
}

func TestDetectCollectors_RPMAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "/usr/bin/rpm", nil },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "linux",
	}

	collectors := detectCollectors(deps)

	if len(collectors) != 1 {
		t.Fatalf("expected 1 collector, got %d", len(collectors))
	}
	if collectors[0].Name() != "rpm" {
		t.Errorf("expected collector name %q, got %q", "rpm", collectors[0].Name())
	}
}

func TestDetectCollectors_NoneAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		snapLookPath:   noSnap(),
		goos:           "linux",
	}

	collectors := detectCollectors(deps)

	if len(collectors) != 0 {
		t.Fatalf("expected 0 collectors, got %d", len(collectors))
	}
}

func TestDetectCollectors_BothAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/dpkg_status_basic",
		rpmLookPath:    func() (string, error) { return "/usr/bin/rpm", nil },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "linux",
	}

	collectors := detectCollectors(deps)

	if len(collectors) != 2 {
		t.Fatalf("expected 2 collectors, got %d", len(collectors))
	}

	names := make(map[string]bool)
	for _, c := range collectors {
		names[c.Name()] = true
	}
	if !names["apt"] {
		t.Error("expected apt collector to be present")
	}
	if !names["rpm"] {
		t.Error("expected rpm collector to be present")
	}
}

func TestDetectCollectors_MacOSAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "darwin",
	}
	collectors := detectCollectors(deps)
	if len(collectors) != 1 {
		t.Fatalf("expected 1 collector, got %d", len(collectors))
	}
	if collectors[0].Name() != "softwareupdate" {
		t.Errorf("expected collector name %q, got %q", "softwareupdate", collectors[0].Name())
	}
}

func TestDetectCollectors_BrewAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "/usr/local/bin/brew", nil },
		goos:           "linux",
	}
	collectors := detectCollectors(deps)
	if len(collectors) != 1 {
		t.Fatalf("expected 1 collector, got %d", len(collectors))
	}
	if collectors[0].Name() != "homebrew" {
		t.Errorf("expected collector name %q, got %q", "homebrew", collectors[0].Name())
	}
}

func TestDetectCollectors_DarwinWithBrew(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "/opt/homebrew/bin/brew", nil },
		goos:           "darwin",
	}
	collectors := detectCollectors(deps)
	if len(collectors) != 2 {
		t.Fatalf("expected 2 collectors, got %d", len(collectors))
	}
	names := make(map[string]bool)
	for _, c := range collectors {
		names[c.Name()] = true
	}
	if !names["softwareupdate"] {
		t.Error("expected softwareupdate collector")
	}
	if !names["homebrew"] {
		t.Error("expected homebrew collector")
	}
}

func TestDetectCollectors_PlatformDetectors(t *testing.T) {
	called := false
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "linux",
		platformDetectors: []collectorDetectorFunc{
			func() packageCollector {
				called = true
				return nil
			},
		},
	}

	collectors := detectCollectors(deps)
	if !called {
		t.Error("platform detector was not called")
	}
	if len(collectors) != 0 {
		t.Fatalf("expected 0 collectors, got %d", len(collectors))
	}
}

func TestDetectCollectors_PlatformDetectorReturnsCollector(t *testing.T) {
	fake := &aptCollector{statusPath: "testdata/dpkg_status_basic"}
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "linux",
		platformDetectors: []collectorDetectorFunc{
			func() packageCollector { return fake },
		},
	}

	collectors := detectCollectors(deps)
	if len(collectors) != 1 {
		t.Fatalf("expected 1 collector, got %d", len(collectors))
	}
	if collectors[0].Name() != "apt" {
		t.Errorf("expected collector name %q, got %q", "apt", collectors[0].Name())
	}
}

func TestDetectCollectors_SnapAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		snapLookPath:   func() (string, error) { return "/usr/bin/snap", nil },
		goos:           "linux",
	}
	collectors := detectCollectors(deps)
	if len(collectors) != 1 {
		t.Fatalf("expected 1 collector, got %d", len(collectors))
	}
	if collectors[0].Name() != "snap" {
		t.Errorf("expected collector name %q, got %q", "snap", collectors[0].Name())
	}
}

func TestDetectCollectors_SnapNotAvailable(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		snapLookPath:   noSnap(),
		goos:           "linux",
	}
	collectors := detectCollectors(deps)
	if len(collectors) != 0 {
		t.Fatalf("expected 0 collectors, got %d", len(collectors))
	}
}

func TestDetectCollectors_SnapNilLookPath(t *testing.T) {
	deps := detectorDeps{
		dpkgStatusPath: "testdata/nonexistent",
		rpmLookPath:    func() (string, error) { return "", exec.ErrNotFound },
		brewLookPath:   func() (string, error) { return "", exec.ErrNotFound },
		goos:           "linux",
		// snapLookPath intentionally nil — should be skipped without panic.
	}
	collectors := detectCollectors(deps)
	if len(collectors) != 0 {
		t.Fatalf("expected 0 collectors, got %d", len(collectors))
	}
}
