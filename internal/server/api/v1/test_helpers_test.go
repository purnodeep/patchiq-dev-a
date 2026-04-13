package v1_test

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Shared pgx.Tx / TxBeginner fakes used by handler tests. These helpers
// used to live in groups_test.go; groups.go and its tests were removed in
// the tags-replace-groups Phase 2 migration but the test machinery they
// defined is still broadly used by workflow, policy, and other handler
// tests that need to exercise transactional paths without a real DB.

// fakeGroupQuerier survives only as an opaque placeholder so existing
// test composition `fakeTx{q: &fakeGroupQuerier{}}` compiles. It does not
// model any group query surface (the groups feature no longer exists).
type fakeGroupQuerier struct{}

// fakeTx implements pgx.Tx for unit tests.
type fakeTx struct {
	q          *fakeGroupQuerier
	queryRowFn func(ctx context.Context, sql string, args ...any) pgx.Row
}

func (f *fakeTx) Begin(_ context.Context) (pgx.Tx, error) { return f, nil }
func (f *fakeTx) Commit(_ context.Context) error          { return nil }
func (f *fakeTx) Rollback(_ context.Context) error        { return nil }
func (f *fakeTx) CopyFrom(_ context.Context, _ pgx.Identifier, _ []string, _ pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (f *fakeTx) SendBatch(_ context.Context, _ *pgx.Batch) pgx.BatchResults { return nil }
func (f *fakeTx) LargeObjects() pgx.LargeObjects                             { return pgx.LargeObjects{} }
func (f *fakeTx) Prepare(_ context.Context, _ string, _ string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (f *fakeTx) Exec(_ context.Context, _ string, _ ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}
func (f *fakeTx) Query(_ context.Context, _ string, _ ...any) (pgx.Rows, error) { return nil, nil }
func (f *fakeTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if f.queryRowFn != nil {
		return f.queryRowFn(ctx, sql, args...)
	}
	return errRow{pgx.ErrNoRows}
}
func (f *fakeTx) Conn() *pgx.Conn { return nil }

// errRow implements pgx.Row, returning a fixed error from Scan.
type errRow struct{ err error }

func (e errRow) Scan(_ ...any) error { return e.err }

// fakeTxBeginner satisfies v1.TxBeginner for unit tests.
type fakeTxBeginner struct {
	tx  *fakeTx
	err error
}

func (f *fakeTxBeginner) Begin(_ context.Context) (pgx.Tx, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.tx, nil
}
