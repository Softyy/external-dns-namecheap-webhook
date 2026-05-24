.PHONY: test lint build docker

test:
	go test -race ./...

lint:
	go vet ./...

build:
	go build -o namecheap-webhook ./cmd/webhook

docker:
	docker build -t namecheap-webhook .

clean:
	rm -f namecheap-webhook