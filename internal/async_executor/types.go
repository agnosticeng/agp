package async_executor

import (
	"time"

	"github.com/agnosticeng/agp/internal/backend"
)

type ResultCompression string

const (
	ResultCompressionNone ResultCompression = ""
	ResultCompressionGZIP ResultCompression = "GZIP"
)

type Status string

const (
	StatusPending   Status = "PENDING"
	StatusRunning   Status = "RUNNING"
	StatusCanceled  Status = "CANCELED"
	StatusFailed    Status = "FAILED"
	StatusSucceeded Status = "SUCCEEDED"
)

type SortBy string

const (
	SortByCreatedAt   SortBy = "CREATED_AT"
	SortByCompletedAt SortBy = "COMPLETED_AT"
)

type ResultMetadata struct {
	Schema             backend.Schema    `json:"schema"`
	NumRows            int64             `json:"num_rows"`
	StoragePath        string            `json:"storage_path"`
	StorageCompression ResultCompression `json:"storage_compression"`
}

type Execution struct {
	Id        int64
	CreatedAt time.Time
	CreatedBy string
	QueryId   string
	QueryHash string
	Query     string
	Tier      string
	Status    Status

	CollapsedCounter int64
	PickedAt         *time.Time
	PickedBy         *string
	Progress         *backend.Progress
	DeadAt           *time.Time
	CompletedAt      *time.Time
	Result           *ResultMetadata
	Error            *string
}

type Lease struct {
	Id        int64
	Key       string
	Owner     string
	EndOfTerm time.Time
}
