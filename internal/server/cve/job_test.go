package cve

import (
	"context"
	"strings"
	"testing"

	"github.com/riverqueue/river"
)

func TestNVDSyncJobArgs_Kind(t *testing.T) {
	args := NVDSyncJobArgs{TenantID: "tenant-1"}
	if got := args.Kind(); got != "cve_nvd_sync" {
		t.Errorf("Kind() = %q, want cve_nvd_sync", got)
	}
}

func TestEndpointMatchJobArgs_Kind(t *testing.T) {
	args := EndpointMatchJobArgs{TenantID: "tenant-1", EndpointID: "ep-1"}
	if got := args.Kind(); got != "cve_endpoint_match" {
		t.Errorf("Kind() = %q, want cve_endpoint_match", got)
	}
}

var _ river.JobArgs = NVDSyncJobArgs{}
var _ river.JobArgs = EndpointMatchJobArgs{}

func TestNVDSyncWorker_Work_NilService(t *testing.T) {
	w := NewNVDSyncWorker(nil)
	err := w.Work(context.Background(), &river.Job[NVDSyncJobArgs]{Args: NVDSyncJobArgs{TenantID: "test"}})
	if err == nil {
		t.Fatal("expected error for nil sync service")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestEndpointMatchWorker_Work_NilMatcher(t *testing.T) {
	w := NewEndpointMatchWorker(nil)
	err := w.Work(context.Background(), &river.Job[EndpointMatchJobArgs]{Args: EndpointMatchJobArgs{TenantID: "test", EndpointID: "ep1"}})
	if err == nil {
		t.Fatal("expected error for nil matcher")
	}
	if !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unexpected error: %v", err)
	}
}
