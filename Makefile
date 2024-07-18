default: identity

.PHONY: clean, test
clean:
	rm -rf ./identity

identity: clean
	go build -o identity ./cmd/identity/. 

dev:
	go run ./cmd/identity 127.0.0.1

test:
	go test -v ./test
