.PHONY: test test-v build run lint fmt tidy check clean

test:
	go test ./...

test-v:
	go test ./... -v

build:
	go build -o reap_test .

run:
	go run .

lint:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

check: fmt lint test

clean:
	rm -f reap_test
