default: all

all: generate lint test

clean:
	rm -f coverage.out

generate: rows_mock_test.go

rows_mock_test.go:
	go install github.com/vektra/mockery/v2@v2.23.1
	go generate ./...

lint:
	go vet ./...
	golangci-lint run

test:
	gotest -coverprofile coverage.out -race -timeout 10s
