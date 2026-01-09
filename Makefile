default: fmt lint install generate

build:
	go build -v ./...

generate docs:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

test-race:
	CGO_ENABLED=1 go test -race -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

.PHONY: fmt lint test test-race testacc build install generate docs
