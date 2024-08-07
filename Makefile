default: identity

.PHONY: clean, test
clean:
	rm -rf ./identity

identity: clean
	go build -o identity ./cmd/identity/. 

dev:
	go run ./*.go 127.0.0.1

test:
	go test -v ./test
