package workers_test

import (
	"context"
	"testing"

	"github.com/skenzeriq/patchiq/internal/server/workers"
)

type mockZitadelClient struct {
	users []workers.ZitadelUser
}

func (m *mockZitadelClient) ListUsers(_ context.Context) ([]workers.ZitadelUser, error) {
	return m.users, nil
}

type mockUserProvisioner struct {
	provisioned []string
}

func (m *mockUserProvisioner) EnsureUser(_ context.Context, u workers.ZitadelUser) error {
	m.provisioned = append(m.provisioned, u.ID)
	return nil
}

func TestUserSync_Run(t *testing.T) {
	client := &mockZitadelClient{
		users: []workers.ZitadelUser{
			{ID: "user-1", Email: "alice@example.com", DisplayName: "Alice"},
			{ID: "user-2", Email: "bob@example.com", DisplayName: "Bob"},
		},
	}
	provisioner := &mockUserProvisioner{}

	syncer := workers.NewUserSyncer(client, provisioner)
	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(provisioner.provisioned) != 2 {
		t.Errorf("provisioned %d users, want 2", len(provisioner.provisioned))
	}
}

func TestUserSync_EmptyList(t *testing.T) {
	client := &mockZitadelClient{users: nil}
	provisioner := &mockUserProvisioner{}

	syncer := workers.NewUserSyncer(client, provisioner)
	err := syncer.Sync(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(provisioner.provisioned) != 0 {
		t.Errorf("provisioned %d users, want 0", len(provisioner.provisioned))
	}
}
