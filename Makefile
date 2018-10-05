default: fmt vet
	@go build -o parrot .
fmt:
	@go fmt ./...
vet:
	@go vet ./...

.PHONY: fmt vet
