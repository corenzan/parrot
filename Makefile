.PHONY: fmt

default: fmt
	@go build .

fmt:
	@go fmt ./...
	@go vet ./...
