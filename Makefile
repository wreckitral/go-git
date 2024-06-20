build:
	@go build -o bin/gg ./cmd/

run: build
	@./bin/gg

test:
	@go test ./... -v
