.PHONY: build clean test

BINARY := obsidian-daily-macos

build:
	CGO_ENABLED=0 go build -o $(BINARY) .

test:
	go test ./...

clean:
	rm -f $(BINARY)
