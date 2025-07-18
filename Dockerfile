FROM golang:1.24-bullseye AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 \
    GOOS=linux \
    go build \
      -ldflags="-s -w" \
      -o /app \
      ./main.go

FROM scratch
COPY --from=builder /app /app
ENTRYPOINT ["/app"]
