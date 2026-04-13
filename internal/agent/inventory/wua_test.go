//go:build windows

package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockSearcher struct {
	updates []windowsUpdate
	err     error
}

func (m *mockSearcher) Search(_ context.Context, _ string) ([]windowsUpdate, error) {
	return m.updates, m.err
}

func TestWUACollector_Name(t *testing.T) {
	c := &wuaCollector{}
	assert.Equal(t, "wua", c.Name())
}

func TestWUACollector_Collect(t *testing.T) {
	c := &wuaCollector{
		searcher: &mockSearcher{
			updates: []windowsUpdate{
				{Title: "2024-02 Cumulative Update", KBID: "KB5034765", Severity: "Critical"},
			},
		},
	}

	pkgs, err := c.Collect(context.Background())
	assert.NoError(t, err)
	assert.Len(t, pkgs, 1)
	assert.Equal(t, "KB5034765", pkgs[0].Name)
	assert.Equal(t, "wua", pkgs[0].Source)
}
