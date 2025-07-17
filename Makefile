PROTO_FILES := $(wildcard proto/*)
BUF_VERSION := 1.54.0

# Buf

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

# Sqlc

.PHONY: sqlc-gen
sqlc-gen:
	go generate -x sqlite/sqlc_gen.go
