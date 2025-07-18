# IoT Metrics

An IoT metrics service exposing both REST and gRPC APIs on port 8080 (default).

- **REST**
  - [OpenAPI spec](openapi.yaml)
  - Implements handlers using [Echo](https://echo.labstack.com)
- **gRPC**
  - [Protobuf schema](proto/iot/v1/service.proto)
  - Implements handlers using [Connect](https://connectrpc.com)
  - Connect handlers support the full gRPC wire format out of the box

## System Design

#### Handlers

- REST and gRPC handlers are implemented separately.
  - 💡️ While I could have unified them under a single Connect handler and used
    [gRPC-Gateway](https://github.com/grpc-ecosystem/grpc-gateway)
    or [Vanguard](https://github.com/connectrpc/vanguard-go) to handle REST ↔ gRPC transcoding, I kept them separate to honor the assignment’s requirements.
- Each handler validates incoming requests and returns standardized, structured
  error responses.

#### Data store

- SQLite is used as the data store, but alternatives (e.g. in-memory, Postgres, etc.) can be supported by simply
  implementing the `device.Repository` interface.

#### Logging

- Requests are automatically logged via middleware. 
- In addition, dedicated logs are captured for:
  - Device configuration updates
  - Recorded device metrics
  - Triggered alerts

#### Alerting

- After a metric is recorded, thresholds are checked and an alert is triggered if any are breached.
- Alerts are triggered synchronously within the `POST /devices/:device_id/metrics` handler.
  - 💡️ For improved performance, alerting could be moved to an asynchronous background task, but this was omitted to
    keep
    the solution simple.

## Running

Configure the app using `config.yaml`.

### Go (1.24)

```shell
make go-run
```

### Docker

```shell
make docker-build
make docker-run
```

## API

Specifications:

- **REST:** [OpenAPI spec](openapi.yaml)
- **gRPC:** [Protobuf schema](proto/iot/v1/service.proto)

### Health

Checks the health status of the server.

- **REST:** `GET /healthz`

  ```shell
  curl -i http://localhost:8080/healthz
  ```

- **gRPC:** `grpc.health.v1.Health/Check`

  ```shell
  grpcurl -plaintext localhost:8080 grpc.health.v1.Health/Check
  ```

### Configure device

Configures device thresholds, replacing any existing configuration (upsert).

- **REST:** `POST /devices/:device_id/config`

  ```shell
  curl -i -X POST http://localhost:8080/devices/d-123/config \
      -H "Content-Type: application/json" \
      -d '{
        "temperature_threshold": 30.85,
        "battery_threshold": 20
      }'
  ```

- **gRPC:** `iot.v1.DeviceService/ConfigureDevice`

  ```shell
  grpcurl -plaintext \
      -d '{
        "device_id":             "d-123",
        "temperature_threshold": 30.85,
        "battery_threshold":     20
      }' \
      localhost:8080 iot.v1.DeviceService/ConfigureDevice
  ```

### Record device metric

Records a device metric and triggers an alert if it breaches configured thresholds.

- **REST:** `POST /devices/:device_id/config`

  ```shell
  curl -i -X POST http://localhost:8080/devices/d-123/metrics \
      -H "Content-Type: application/json" \
      -d '{
        "timestamp":  "2025-07-17T12:00:00Z",
        "temperature": 40.50,
        "battery":     10
      }'
  ```

- **gRPC:** `iot.v1.DeviceService/ConfigureDevice`

  ```shell
  grpcurl -plaintext \
      -d '{
        "device_id":  "d-123",
        "timestamp":  "2025-07-17T12:00:00Z",
        "temperature": 40.50,
        "battery":     10
      }' \
      localhost:8080 iot.v1.DeviceService/RecordMetric
  ```

### Get device alerts

Retrieves recent device alerts with support for timeframe filtering and cursor based pagination.

- **REST:** `POST /devices/:device_id/alerts`
  - Query params:

    | Name              | Example                                                               |
    |-------------------|-----------------------------------------------------------------------|
    | `timeframe.start` | 2025-07-16T12:00:00Z                                                  |
    | `timeframe.end`   | 2025-07-18T12:00:00Z                                                  |
    | `page.size`       | 5 (default: 100)                                                      |
    | `page.token`      | `eyJMYXN0VGltZSI6IjIwMjUtMDQtMjVUMTI6MDA6MDBaIiwiTGFzdElEIjoxMTg5M30` |

  ```shell
  curl -i -H "Accept: application/json" "http://localhost:8080/devices/d-123/alerts?"\
  "timeframe.start=2025-07-16T12:00:00Z&"\
  "timeframe.end=2025-07-18T12:00:00Z&"\
  "page.size=5"
  ```

- **gRPC:** `iot.v1.DeviceService/GetDeviceAlerts`

  ```shell
  grpcurl -plaintext \
    -d '{
      "device_id":  "d-123",
      "timeframe":  {
        "start": "2025-07-16T12:00:00Z",
        "end":   "2025-07-18T12:00:00Z"
      },
      "page_size":  5,
      "page_token": ""
    }' \
    localhost:8080 iot.v1.DeviceService/GetDeviceAlerts
  ```

## Bonus Tasks

### Device rate limiting

- Device level rate limiting is enabled via `config.yaml` (currently 5 requests every second).
- It can be disabled by commenting out the `deviceRateLimit` section in `config.yaml`.
- The rate limiter is implemented as a middleware and allows each device to make up to `deviceRateLimit.tokens`
requests per `deviceRateLimit.seconds` seconds, across all APIs.

### Simulate 1000+ concurrent devices

The `stress` CLI tool in `cmd/stress/` can be used to start a configurable worker pool that concurrently sends
`RecordMetric` requests for `n` randomly generated devices.

Requests are made on startup to configure each device with the following thresholds:

- Temperature threshold: `50.00`
- Battery threshold: `50`

For each recorded metric, temperature and battery values `v` are randomly generated such that `0` <= `v` <=
`threshold * 2`.

This means there is a 50% chance that a recorded value will exceed its threshold, triggering an alert.

Run the command below for usage instructions (requires Go >= 1.24):

````shell
go run ./cmd/stress --help
````

Example for 1000 devices concurrently recording metrics at 100ms intervals:

```shell
go run ./cmd/stress --devices 1000 --interval 100
```