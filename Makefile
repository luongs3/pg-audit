.PHONY: build test run lint clean tidy

BIN := bin/pg-audit

build:
	go build -ldflags "-X main.version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o $(BIN) ./cmd/pg-audit

tidy:
	go mod tidy

test:
	go test ./...

run: build
	./$(BIN) run --dsn "$(PGAUDIT_DSN)"

lint:
	go vet ./...
	gofmt -l . | tee /dev/stderr | (! read)

clean:
	rm -rf bin
