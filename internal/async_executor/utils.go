package async_executor

import (
	"fmt"
	"hash/fnv"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func fnv1aHashInt64Sum(s string) int64 {
	var h = fnv.New64a()
	h.Write([]byte(s))
	return int64(h.Sum64())
}

func countRows(rows pgx.Rows) (int, error) {
	var count int
	defer rows.Close()

	for rows.Next() {
		count++
	}

	return count, rows.Err()
}

func (aex *AsyncExecutor) buildResultURL(ex *Execution) (*url.URL, string, error) {
	var (
		u, err = url.Parse(aex.conf.ResultStoragePrefix)
		path   = strconv.FormatInt(ex.Id, 10)
	)

	if err != nil {
		return nil, "", fmt.Errorf("failed to build result url: %w", err)
	}

	u.Path = filepath.Join(u.Path, path)
	return u, path, nil
}
