package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/skenzeriq/patchiq/internal/agent/api"
)

func TestSettingsHandler(t *testing.T) {
	info := api.SettingsInfo{
		AgentVersion: "1.0.0",
		ConfigFile:   "/etc/patchiq/agent.yaml",
		DataDir:      "/var/lib/patchiq",
		ServerURL:    "grpc.example.com:50051",
		ScanInterval: "6h",
		AutoDeploy:   false,
	}
	h := api.NewSettingsHandler(api.StaticSettingsProvider(info))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/settings", nil)
	w := httptest.NewRecorder()
	h.Get(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	var result api.SettingsInfo
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.AgentVersion != "1.0.0" {
		t.Errorf("want 1.0.0, got %s", result.AgentVersion)
	}
	if result.ConfigFile != "/etc/patchiq/agent.yaml" {
		t.Errorf("want config file, got %s", result.ConfigFile)
	}
	if result.ScanInterval != "6h" {
		t.Errorf("want 6h, got %s", result.ScanInterval)
	}
}

func TestStaticSettingsProvider(t *testing.T) {
	info := api.SettingsInfo{AgentVersion: "2.0.0", AutoDeploy: true}
	p := api.StaticSettingsProvider(info)
	got := p.Settings()
	if got.AgentVersion != "2.0.0" {
		t.Errorf("want 2.0.0, got %s", got.AgentVersion)
	}
	if !got.AutoDeploy {
		t.Error("want AutoDeploy=true")
	}
}
