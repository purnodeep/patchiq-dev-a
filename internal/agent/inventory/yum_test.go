package inventory

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
)

// fakeRunner implements commandRunner for testing.
type fakeRunner struct {
	output []byte
	err    error
}

func (f fakeRunner) Run(_ context.Context, _ string, _ ...string) ([]byte, error) {
	return f.output, f.err
}

func TestRPMCollector_Name(t *testing.T) {
	c := &rpmCollector{runner: fakeRunner{}}
	if got := c.Name(); got != "rpm" {
		t.Errorf("Name() = %q, want %q", got, "rpm")
	}
}

func TestParseRPMOutput_Basic(t *testing.T) {
	data, err := os.ReadFile("testdata/rpm_output_basic")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs := parseRPMOutput(data)

	if len(pkgs) != 5 {
		t.Fatalf("expected 5 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}

	first := pkgs[0]
	if first.Name != "bash" {
		t.Errorf("first package Name = %q, want %q", first.Name, "bash")
	}
	if first.Version != "5.1.8" {
		t.Errorf("first package Version = %q, want %q", first.Version, "5.1.8")
	}
	if first.Release != "1.el9" {
		t.Errorf("first package Release = %q, want %q", first.Release, "1.el9")
	}
	if first.Architecture != "x86_64" {
		t.Errorf("first package Architecture = %q, want %q", first.Architecture, "x86_64")
	}
	if first.Source != "rpm" {
		t.Errorf("first package Source = %q, want %q", first.Source, "rpm")
	}
}

func TestParseRPMOutput_EdgeCases(t *testing.T) {
	data, err := os.ReadFile("testdata/rpm_output_edge_cases")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	pkgs := parseRPMOutput(data)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}

	if pkgs[0].Name != "valid-pkg" {
		t.Errorf("first valid package Name = %q, want %q", pkgs[0].Name, "valid-pkg")
	}
	if pkgs[1].Name != "valid-after-bad" {
		t.Errorf("second valid package Name = %q, want %q", pkgs[1].Name, "valid-after-bad")
	}
}

func TestParseRPMOutput_EmptyInput(t *testing.T) {
	pkgs := parseRPMOutput([]byte{})
	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestRPMCollector_Collect_Success(t *testing.T) {
	data, err := os.ReadFile("testdata/rpm_output_basic")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	c := &rpmCollector{runner: fakeRunner{output: data}}
	pkgs, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}

	if len(pkgs) != 5 {
		t.Fatalf("expected 5 packages, got %d: %v", len(pkgs), packageNames(pkgs))
	}
}

func TestRPMCollector_Collect_CommandFailure(t *testing.T) {
	c := &rpmCollector{runner: fakeRunner{err: errors.New("command not found")}}

	_, err := c.Collect(context.Background())
	if err == nil {
		t.Fatal("expected error from Collect, got nil")
	}
}

func BenchmarkParseRPMOutput_1000Packages(b *testing.B) {
	var buf bytes.Buffer
	for i := range 1000 {
		fmt.Fprintf(&buf, "pkg-%d\t%d.0.0\t1.el9\tx86_64\n", i, i)
	}
	data := buf.Bytes()

	b.ResetTimer()
	for range b.N {
		pkgs := parseRPMOutput(data)
		if len(pkgs) != 1000 {
			b.Fatalf("expected 1000, got %d", len(pkgs))
		}
	}
}
