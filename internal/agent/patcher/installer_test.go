package patcher

import (
	"testing"
)

func TestDetectInstaller_apt(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "/usr/bin/apt-get", nil },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
	}
	installer := detectInstaller(deps, nil, nil)
	if installer == nil {
		t.Fatal("expected non-nil installer")
	}
	if installer.Name() != "apt" {
		t.Errorf("name = %q, want %q", installer.Name(), "apt")
	}
}

func TestDetectInstaller_dnf(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "/usr/bin/dnf", nil },
		yumLookPath:    func() (string, error) { return "/usr/bin/yum", nil },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
	}
	installer := detectInstaller(deps, nil, nil)
	if installer == nil {
		t.Fatal("expected non-nil installer")
	}
	if installer.Name() != "dnf" {
		t.Errorf("name = %q, want %q", installer.Name(), "dnf")
	}
}

func TestDetectInstaller_yum(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "/usr/bin/yum", nil },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
	}
	installer := detectInstaller(deps, nil, nil)
	if installer == nil {
		t.Fatal("expected non-nil installer")
	}
	if installer.Name() != "yum" {
		t.Errorf("name = %q, want %q", installer.Name(), "yum")
	}
}

func TestDetectInstaller_none(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
	}
	installer := detectInstaller(deps, nil, nil)
	if installer != nil {
		t.Errorf("expected nil installer, got %q", installer.Name())
	}
}

func TestDetectInstallers_Multiple(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "/usr/bin/apt-get", nil },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "/usr/local/bin/brew", nil },
		goos:           "linux",
	}
	installers := detectInstallers(deps, nil, nil)
	if len(installers) != 2 {
		t.Fatalf("expected 2 installers, got %d", len(installers))
	}
	if installers[0].Name() != "apt" {
		t.Errorf("first installer = %q, want apt", installers[0].Name())
	}
	if installers[1].Name() != "homebrew" {
		t.Errorf("second installer = %q, want homebrew", installers[1].Name())
	}
}

func TestDetectInstallers_None(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
	}
	installers := detectInstallers(deps, nil, nil)
	if len(installers) != 0 {
		t.Fatalf("expected 0 installers, got %d", len(installers))
	}
}

func TestDetectInstallers_AptAndDnf(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "/usr/bin/apt-get", nil },
		dnfLookPath:    func() (string, error) { return "/usr/bin/dnf", nil },
		yumLookPath:    func() (string, error) { return "/usr/bin/yum", nil },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
	}
	installers := detectInstallers(deps, nil, nil)
	if len(installers) != 2 {
		t.Fatalf("expected 2 installers, got %d", len(installers))
	}
	if installers[0].Name() != "apt" {
		t.Errorf("first installer = %q, want apt", installers[0].Name())
	}
	if installers[1].Name() != "dnf" {
		t.Errorf("second installer = %q, want dnf", installers[1].Name())
	}
}

func TestDetectInstallers_DarwinAll(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "/opt/homebrew/bin/brew", nil },
		goos:           "darwin",
	}
	installers := detectInstallers(deps, nil, nil)
	if len(installers) != 2 {
		t.Fatalf("expected 2 installers, got %d", len(installers))
	}
	if installers[0].Name() != "softwareupdate" {
		t.Errorf("first installer = %q, want softwareupdate", installers[0].Name())
	}
	if installers[1].Name() != "homebrew" {
		t.Errorf("second installer = %q, want homebrew", installers[1].Name())
	}
}

func TestDetectInstaller_PlatformDetectors(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
		platformDetectors: []installerDetectorFunc{
			func(executor CommandExecutor) Installer {
				return &msiInstaller{executor: executor}
			},
		},
	}
	installer := detectInstaller(deps, nil, nil)
	if installer == nil {
		t.Fatal("expected non-nil installer from platform detector")
	}
	if installer.Name() != "msi" {
		t.Errorf("name = %q, want %q", installer.Name(), "msi")
	}
}

func TestDetectInstaller_PlatformDetectorsNilSkipped(t *testing.T) {
	deps := installerDetectorDeps{
		aptGetLookPath: func() (string, error) { return "", errNotFound },
		dnfLookPath:    func() (string, error) { return "", errNotFound },
		yumLookPath:    func() (string, error) { return "", errNotFound },
		brewLookPath:   func() (string, error) { return "", errNotFound },
		goos:           "linux",
		platformDetectors: []installerDetectorFunc{
			func(_ CommandExecutor) Installer { return nil },
		},
	}
	installer := detectInstaller(deps, nil, nil)
	if installer != nil {
		t.Errorf("expected nil installer, got %q", installer.Name())
	}
}
