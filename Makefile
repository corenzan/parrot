parrot: fmt vet
	@go build -o parrot .
run: fmt vet
	@go run main.go
test: fmt vet
	@go test ./...
fmt:
	@go fmt ./...
vet:
	@go vet ./...
.PHONY: run test fmt vet 
