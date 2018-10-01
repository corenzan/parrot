default: fmt vet
	@go build -o dist/parrot .
fmt:
	@go fmt ./...
vet:
	@go vet ./...

.PHONY: fmt vet
