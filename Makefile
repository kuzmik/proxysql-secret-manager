SHELL := /bin/bash

# The name of the executable
TARGET := 'proxysql-secret-manager'

# Use linker flags to provide version/build settings to the target.
LDFLAGS=-ldflags "-s -w"

all: clean lint build

$(TARGET):
	@go build $(LDFLAGS) -o $(TARGET) .

build: clean $(TARGET)
	@true

run: build
	@./$(TARGET)

clean:
	@rm -rf $(TARGET) *.test *.out tmp/* coverage dist

lint:
	@gofumpt -l -w .
	@go vet ./...
	@golangci-lint run --config=.golangci.yml --allow-parallel-runners

test:
	@mkdir -p coverage
	@go test ./... --shuffle=on --coverprofile coverage/coverage.out

coverage: test
	@go tool cover -html=coverage/coverage.out
