BINARY := ratatoskr
MAIN := ./cmd/ratatoskr
DIST := dist
VERSION ?= v1.0.0

.PHONY: test dist clean

test:
	go test ./...
	git diff --check

dist:
	rm -rf $(DIST)
	mkdir -p $(DIST)
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY)_$(VERSION)_darwin_arm64 $(MAIN)
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY)_$(VERSION)_darwin_amd64 $(MAIN)
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY)_$(VERSION)_linux_amd64 $(MAIN)
	GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(DIST)/$(BINARY)_$(VERSION)_linux_arm64 $(MAIN)
	cd $(DIST) && shasum -a 256 * > checksums.txt

clean:
	rm -rf $(DIST) $(BINARY)
