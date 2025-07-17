-- name: SaveDeviceMetric :exec
INSERT INTO metrics (device_id, temperature, battery, timestamp)
VALUES (?, ?, ?, ?);

-- name: GetDeviceMetrics :many
SELECT *
FROM metrics
WHERE device_id = :device_id
  -- time window
  AND (CAST(sqlc.narg('start_ts') AS INTEGER) IS NULL OR timestamp >= sqlc.narg('start_ts'))
  AND (CAST(sqlc.narg('end_ts') AS INTEGER) IS NULL OR timestamp <= sqlc.narg('end_ts'))
  -- composite cursor
  AND (
    CAST(sqlc.narg('last_ts') AS INTEGER) IS NULL
        OR (
        -- timestamp less than previous page last row
        timestamp < sqlc.narg('last_ts')
            OR (
            -- or timestamp equal to previous page last row
            timestamp = sqlc.narg('last_ts')
                -- but is less than last row id
                AND (CAST(sqlc.narg('last_id') AS INTEGER) IS NULL OR id < sqlc.narg('last_id'))
            )
        )
    )
ORDER BY timestamp DESC, id DESC
LIMIT :limit;

-- name: UpsertDeviceConfig :exec
INSERT INTO configs (device_id, temperature_threshold, battery_threshold)
VALUES (?, ?, ?)
ON CONFLICT(device_id) DO UPDATE
    SET temperature_threshold=excluded.temperature_threshold,
        battery_threshold=excluded.battery_threshold;

-- name: GetDeviceConfig :one
SELECT temperature_threshold, battery_threshold
FROM configs
WHERE device_id = ?;

-- name: SaveDeviceAlert :exec
INSERT INTO alerts (device_id, reason, desc, timestamp)
VALUES (?, ?, ?, ?);

-- name: GetDeviceAlerts :many
SELECT *
FROM alerts
WHERE device_id = :device_id
  -- time window
  AND (CAST(sqlc.narg('start_ts') AS INTEGER) IS NULL OR timestamp >= sqlc.narg('start_ts'))
  AND (CAST(sqlc.narg('end_ts') AS INTEGER) IS NULL OR timestamp <= sqlc.narg('end_ts'))
  -- composite cursor
  AND (
    CAST(sqlc.narg('last_ts') AS INTEGER) IS NULL
        OR (
        -- timestamp less than previous page last row
        timestamp < sqlc.narg('last_ts')
            OR (
            -- or timestamp equal to previous page last row
            timestamp = sqlc.narg('last_ts')
                -- but is less than last row id
                AND (CAST(sqlc.narg('last_id') AS INTEGER) IS NULL OR id < sqlc.narg('last_id'))
            )
        )
    )
ORDER BY timestamp DESC, id DESC
LIMIT :limit;