.PHONY: help build build-web build-go test test-go test-web lint run clean gen-api

help:
	@echo "Available targets:"
	@echo "  build       — build web SPA and Go binary"
	@echo "  build-web   — build the SvelteKit SPA into internal/webui/dist"
	@echo "  build-go    — build the Go binary (requires build-web first)"
	@echo "  test        — run all tests"
	@echo "  run         — build and run mylib against ./testdata"
	@echo "  gen-api     — regenerate web/src/lib/api/schema.d.ts from running server"
	@echo "  clean       — remove build artifacts"

build: build-web build-go

build-web:
	pnpm --dir web install --frozen-lockfile
	pnpm --dir web build

build-go:
	mkdir -p bin
	go build -o bin/mylib ./cmd/mylib

test: test-go test-web

test-go:
	go test ./...

test-web:
	pnpm --dir web check

lint:
	go vet ./...

run: build
	MYLIB_LIBRARY_ROOTS=$${MYLIB_LIBRARY_ROOTS:-./testdata} \
	MYLIB_DATA_DIR=$${MYLIB_DATA_DIR:-./data} \
	./bin/mylib

gen-api:
	pnpm --dir web gen:api

clean:
	rm -rf bin/ data/ internal/webui/dist/* web/build/ web/.svelte-kit/
	touch internal/webui/dist/.gitkeep
