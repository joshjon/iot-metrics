package sqlite

import (
	"context"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/joshjon/iot-metrics/device"
	"github.com/joshjon/iot-metrics/sqlite/migrations"
)

func TestDeviceRepository_UpsertGetDeviceConfig(t *testing.T) {
	ctx := t.Context()
	repo := newRepo(t, ctx)

	deviceID := "foo"
	cfg := device.Config{
		TemperatureThreshold: 5.55,
		BatteryThreshold:     5,
	}
	err := repo.UpsertDeviceConfig(ctx, deviceID, cfg)
	require.NoError(t, err)

	gotCfg, err := repo.GetDeviceConfig(ctx, deviceID)
	require.NoError(t, err)
	require.Equal(t, cfg, gotCfg)

	_, err = repo.GetDeviceConfig(ctx, "not_exists")
	require.ErrorIs(t, err, device.ErrRepoItemNotFound)
}

func TestDeviceRepository_SaveGetDeviceMetrics(t *testing.T) {
	ctx := t.Context()
	repo := newRepo(t, ctx)
	deviceID := "foo"

	start := time.Now().UTC().Truncate(time.Second)
	middle := start.Add(time.Second)
	end := middle.Add(time.Second)
	count := 12
	saved := make([]device.Metric, count)

	for i := 0; i < count; i++ {
		metric := device.Metric{
			Temperature: float64(i),
			Battery:     int32(i),
			Time:        middle,
		}
		// first and last outside timeframe
		switch i {
		case 0:
			metric.Time = start.Add(-time.Second)
		case count - 1:
			metric.Time = end.Add(time.Second)
		}
		err := repo.SaveDeviceMetric(ctx, deviceID, metric)
		require.NoError(t, err)
		saved[i] = metric
	}

	// expect metrics without first and last
	wantItems := saved[1 : len(saved)-1]

	size := 5
	timeframe := device.Timeframe{Start: &start, End: &end}
	p1, err := repo.GetDeviceMetrics(ctx, deviceID, timeframe, device.RepositoryPageOptions{Size: size})
	require.NoError(t, err)
	require.Len(t, p1.Items, size)

	wantP1Items := wantItems[size:]
	wantP2Items := wantItems[:size]

	slices.Reverse(wantP1Items) // expect to descend by timestamp
	require.Equal(t, wantP1Items, p1.Items)

	require.NotNil(t, p1.NextPageToken)
	require.Positive(t, *p1.NextPageToken.LastID)
	require.Equal(t, wantP1Items[len(wantP1Items)-1].Time, *p1.NextPageToken.LastTime)

	p2, err := repo.GetDeviceMetrics(ctx, deviceID, timeframe, device.RepositoryPageOptions{
		Size:  size,
		Token: p1.NextPageToken,
	})
	require.NoError(t, err)
	require.Len(t, p2.Items, size)

	slices.Reverse(wantP2Items) // expect to descend by timestamp
	require.Equal(t, wantP2Items, p2.Items)

	require.Nil(t, p2.NextPageToken) // no more pages
}

func TestDeviceRepository_SaveGetDeviceAlerts(t *testing.T) {
	ctx := t.Context()
	repo := newRepo(t, ctx)
	deviceID := "foo"

	start := time.Now().UTC().Truncate(time.Second)
	middle := start.Add(time.Second)
	end := middle.Add(time.Second)
	count := 12
	saved := make([]device.Alert, count)

	for i := 0; i < count; i++ {
		alert := device.Alert{
			Reason: device.AlertReasonBatteryLow,
			Desc:   "desc " + strconv.Itoa(i),
			Time:   middle,
		}
		// first and last outside timeframe
		switch i {
		case 0:
			alert.Time = start.Add(-time.Second)
		case count - 1:
			alert.Time = end.Add(time.Second)
		}
		err := repo.SaveDeviceAlert(ctx, deviceID, alert)
		require.NoError(t, err)
		saved[i] = alert
	}

	// expect alerts without first and last
	wantItems := saved[1 : len(saved)-1]

	size := 5
	timeframe := device.Timeframe{Start: &start, End: &end}
	p1, err := repo.GetDeviceAlerts(ctx, deviceID, timeframe, device.RepositoryPageOptions{Size: size})
	require.NoError(t, err)
	require.Len(t, p1.Items, size)

	wantP1Items := wantItems[size:]
	wantP2Items := wantItems[:size]

	slices.Reverse(wantP1Items) // expect to descend by timestamp
	require.Equal(t, wantP1Items, p1.Items)

	require.NotNil(t, p1.NextPageToken)
	require.Positive(t, *p1.NextPageToken.LastID)
	require.Equal(t, wantP1Items[len(wantP1Items)-1].Time, *p1.NextPageToken.LastTime)

	p2, err := repo.GetDeviceAlerts(ctx, deviceID, timeframe, device.RepositoryPageOptions{
		Size:  size,
		Token: p1.NextPageToken,
	})
	require.NoError(t, err)
	require.Len(t, p2.Items, size)

	slices.Reverse(wantP2Items) // expect to descend by timestamp
	require.Equal(t, wantP2Items, p2.Items)

	require.Nil(t, p2.NextPageToken) // no more pages
}

func newRepo(t *testing.T, ctx context.Context) *DeviceRepository {
	db, err := Open(ctx, WithDir(t.TempDir()))
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, db.Close())
	})
	err = Migrate(db, migrations.FS())
	require.NoError(t, err)
	repo := NewDeviceRepository(db)
	return repo
}
