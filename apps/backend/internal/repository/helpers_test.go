package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"testing"

	"github.com/kisnikita/safe-disputes/backend/pkg/log"
	"go.uber.org/zap"
)

type noopLogger struct{}

func (noopLogger) Debug(string, ...zap.Field) {}
func (noopLogger) Info(string, ...zap.Field)  {}
func (noopLogger) Error(string, ...zap.Field) {}
func (noopLogger) Fatal(string, ...zap.Field) {}
func (noopLogger) With(...zap.Field) log.Logger {
	return noopLogger{}
}
func (noopLogger) Sync() error { return nil }

type stubDB struct {
	queryFn func(query string, args []driver.NamedValue) (driver.Rows, error)
	execFn  func(query string, args []driver.NamedValue) (driver.Result, error)
}

type stubConnector struct{ stub *stubDB }

func (c *stubConnector) Connect(context.Context) (driver.Conn, error) { return &stubConn{stub: c.stub}, nil }
func (c *stubConnector) Driver() driver.Driver                         { return &stubDriver{stub: c.stub} }

type stubDriver struct{ stub *stubDB }

func (d *stubDriver) Open(string) (driver.Conn, error) { return &stubConn{stub: d.stub}, nil }

type stubConn struct{ stub *stubDB }

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("not supported") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error)           { return nil, fmt.Errorf("not supported") }

func (c *stubConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.stub.queryFn == nil {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	return c.stub.queryFn(query, args)
}

func (c *stubConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.stub.execFn == nil {
		return nil, fmt.Errorf("unexpected exec: %s", query)
	}
	return c.stub.execFn(query, args)
}

type stubRows struct {
	columns []string
	data    [][]driver.Value
	idx     int
}

func (r *stubRows) Columns() []string { return r.columns }
func (r *stubRows) Close() error      { return nil }
func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.data) {
		return io.EOF
	}
	for i := range dest {
		dest[i] = r.data[r.idx][i]
	}
	r.idx++
	return nil
}

func newRows(columns []string, data ...[]driver.Value) driver.Rows {
	return &stubRows{columns: columns, data: data}
}

func newTestRepo(t *testing.T, stub *stubDB) *Repository {
	t.Helper()
	db := sql.OpenDB(&stubConnector{stub: stub})
	t.Cleanup(func() { _ = db.Close() })
	return &Repository{db: db, logger: noopLogger{}}
}
