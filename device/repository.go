package device

//go:generate go tool moq -out repository_moq_test.go . Repository

import (
	"context"
	"time"
)

const ErrRepoItemNotFound repoErr = "not found"

type repoErr string

func (e repoErr) Error() string { return string(e) }

type RepositoryPageOptions struct {
	Size  int
	Token *RepositoryPageToken
}

type RepositoryPageToken struct {
	LastTime *time.Time
	LastID   *int64
}

type RepositoryPage[T any] struct {
	Items         []T
	NextPageToken *RepositoryPageToken
}

type Repository interface {
	UpsertDeviceConfig(ctx context.Context, deviceID string, config Config) error
	SaveDeviceMetric(ctx context.Context, deviceID string, metric Metric) error
	GetDeviceMetrics(ctx context.Context, deviceID string, timeframe Timeframe, pageOpts RepositoryPageOptions) (RepositoryPage[Metric], error)
	GetDeviceConfig(ctx context.Context, deviceID string) (Config, error)
	SaveDeviceAlert(ctx context.Context, deviceID string, alert Alert) error
	GetDeviceAlerts(ctx context.Context, deviceID string, timeframe Timeframe, pageOpts RepositoryPageOptions) (RepositoryPage[Alert], error)
}
