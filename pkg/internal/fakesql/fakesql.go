// Package fakesql は、クエリごとに用意した結果を返す database/sql のテストダブルを提供する。
//
// 実際のデータベースに接続せずに、 *sql.DB (*sql.Rows) を受け取る関数をテストするために使う。
// 対応しているのはトランザクションを使わない QueryContext のみ。
package fakesql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
)

var (
	// ErrNotSupported は fakesql が対応していない操作を表す
	ErrNotSupported = errors.New("fakesql: not supported")
	// ErrNoRowsPrepared は QueryFunc が結果を返さなかったクエリを表す
	ErrNoRowsPrepared = errors.New("fakesql: no rows prepared for the query")
)

// Rows はクエリ 1 回分の結果
type Rows struct {
	Columns []string
	Values  [][]driver.Value
}

// QueryFunc はクエリと引数から結果を返す。
// エラーを返した場合、そのエラーが QueryContext の呼び出し元に返る。
type QueryFunc func(query string, args []driver.NamedValue) (*Rows, error)

// OpenDB は queryFunc で問い合わせに応答する *sql.DB を返す
func OpenDB(queryFunc QueryFunc) *sql.DB {
	return sql.OpenDB(&connector{queryFunc: queryFunc})
}

type connector struct {
	queryFunc QueryFunc
}

func (c *connector) Connect(context.Context) (driver.Conn, error) {
	return &conn{queryFunc: c.queryFunc}, nil
}

func (c *connector) Driver() driver.Driver {
	return fakeDriver{}
}

// fakeDriver は driver.Connector の要件を満たすためだけに存在する。
// sql.OpenDB は Connector.Connect で接続するため Open は呼ばれない。
type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) {
	return nil, ErrNotSupported
}

type conn struct {
	queryFunc QueryFunc
}

func (c *conn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	r, err := c.queryFunc(query, args)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrNoRowsPrepared
	}
	return &rows{columns: r.Columns, values: r.Values}, nil
}

func (c *conn) Prepare(string) (driver.Stmt, error) {
	return nil, ErrNotSupported
}

func (c *conn) Close() error { return nil }

func (c *conn) Begin() (driver.Tx, error) {
	return nil, ErrNotSupported
}

type rows struct {
	columns []string
	values  [][]driver.Value
	next    int
}

func (r *rows) Columns() []string { return r.columns }

func (r *rows) Close() error { return nil }

func (r *rows) Next(dest []driver.Value) error {
	if r.next >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.next])
	r.next++
	return nil
}
