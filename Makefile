build:
	@go build -o bin/gg

run: build
	@./bin/gg

test:
	@go test ./... -v
