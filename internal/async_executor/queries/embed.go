package queries

import (
	"bytes"
	"context"
	"embed"
	_ "embed"
	"fmt"
	"text/template"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

//go:embed *.sql
var queriesFS embed.FS
var queriesTmpl *template.Template

func init() {
	tmpl, err := template.New("queries").ParseFS(queriesFS, "*.sql")

	if err != nil {
		panic(fmt.Errorf("failed to parse SQL template files: %w", err))
	}

	queriesTmpl = tmpl
}

type Conn interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, sql string, arguments ...any) (pgx.Rows, error)
}

func Exec(
	ctx context.Context,
	tx Conn,
	queryName string,
	params map[string]any,
) (pgconn.CommandTag, error) {
	var buf bytes.Buffer

	if err := queriesTmpl.ExecuteTemplate(&buf, queryName, params); err != nil {
		return pgconn.CommandTag{}, err
	}

	return tx.Exec(ctx, buf.String(), pgx.NamedArgs(params))
}

func Query(
	ctx context.Context,
	tx Conn,
	queryName string,
	params map[string]any,
) (pgx.Rows, error) {
	var buf bytes.Buffer

	if err := queriesTmpl.ExecuteTemplate(&buf, queryName, params); err != nil {
		return nil, err
	}

	return tx.Query(ctx, buf.String(), pgx.NamedArgs(params))
}
