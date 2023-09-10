#!make
BINARY_NAME=hetzner-dyndns
OUTPUT_DIR=/extract
COVERAGE_DIR=/coverage

-include .env

clean:
	go clean
	rm ${OUTPUT_DIR}/${BINARY_NAME}-*

dep:
	go mod download -x

test: dep
	ZONE=${ZONE} TOKEN=${TOKEN} go test -coverprofile=${COVERAGE_DIR}/coverage.out ./...

build-amd64: dep
	CGO_ENABLED=0 GOARCH=amd64 go build -o ${OUTPUT_DIR}/${BINARY_NAME}-amd64 cmd/${BINARY_NAME}/main.go

build-arm64: dep
	CGO_ENABLED=0 GOARCH=arm64 go build -o ${OUTPUT_DIR}/${BINARY_NAME}-arm64 cmd/${BINARY_NAME}/main.go

build: build-amd64 build-arm64

ci: test build

run: dep
	ZONE=${ZONE} TOKEN=${TOKEN} go run cmd/${BINARY_NAME}/main.go