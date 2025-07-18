PROTO_FILES := $(wildcard proto/*)
BUF_VERSION := 1.54.0

# -- Run -----------------------------------------------------------------------

.PHONY: go-run
go-run:
	go run . --config-file config.yaml

# -- Docker --------------------------------------------------------------------

.PHONY: docker-build
docker-build:
	docker build -t local/iot-metrics .

.PHONY: docker-run
docker-run:
	docker run --rm -v "$$(pwd)/config.yaml":/etc/app/config.yaml -p 8080:8080 \
		--name iot-metrics local/iot-metrics \
		--config-file=/etc/app/config.yaml

# -- Buf -----------------------------------------------------------------------

.PHONY: buf-dep-update
buf-dep-update: $(PROTO_FILES)
	docker run -v $$(pwd):/srv -w /srv bufbuild/buf:$(BUF_VERSION) dep update ./proto/

.PHONY: buf-format
buf-format: $(PROTO_FILES)
	docker run -v $$(pwd):/srv -w /srv bufbuild/buf:$(BUF_VERSION) format -w

.PHONY: buf-lint
buf-lint: $(PROTO_FILES)
	docker run -v $$(pwd):/srv -w /srv bufbuild/buf:$(BUF_VERSION) lint

.PHONY: buf-gen
buf-gen: buf-dep-update buf-format buf-lint $(PROTO_FILES)
	rm -rf proto/gen/
	docker run -v $$(pwd):/srv -w /srv bufbuild/buf:$(BUF_VERSION) generate

# -- Sqlc ----------------------------------------------------------------------

.PHONY: sqlc-gen
sqlc-gen:
	go generate -x sqlite/sqlc_gen.go
