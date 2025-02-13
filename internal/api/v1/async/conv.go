package async

import (
	v1 "github.com/agnosticeng/agp/internal/api/v1"
	"github.com/agnosticeng/agp/internal/async_executor"
)

func ToResultMetadata(md *async_executor.ResultMetadata) *ResultMetadata {
	if md == nil {
		return nil
	}

	var (
		res        ResultMetadata
		meta       []v1.Column
		durationMs = md.Duration.Milliseconds()
	)

	for _, column := range md.Schema {
		meta = append(meta, v1.Column(column))
	}

	res.Rows = &md.NumRows
	res.Meta = &meta
	res.Duration = &durationMs
	return &res
}

func ToExecution(ex *async_executor.Execution) *Execution {
	if ex == nil {
		return nil
	}

	var res Execution

	res.Id = ex.Id
	res.QueryId = ex.QueryId
	res.QueryHash = ex.QueryHash
	res.CreatedAt = ex.CreatedAt
	res.Query = ex.Query
	res.Status = ExecutionStatus(ex.Status)
	res.PickedAt = ex.PickedAt
	res.CompletedAt = ex.CompletedAt
	res.Error = ex.Error

	if ex.Progress != nil {
		res.Progress = v1.ToProgress(ex.Progress)
	}

	if ex.Result != nil {
		res.Result = ToResultMetadata(ex.Result)
	}

	return &res
}
