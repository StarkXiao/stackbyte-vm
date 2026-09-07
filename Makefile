APP := stackbyte
VERSION ?= dev
GOFILES := $(shell find . -name '*.go' -not -name '*_test.go' -not -path './dist/*')

.PHONY: fmt test check build package clean

fmt:
	sh scripts/format-go.sh

test:
	go test ./...

check:
	test "$$(find . -name '*.go' -not -name '*_test.go' | wc -l | tr -d ' ')" -gt 20
	test "$$(find . -name '*.go' -not -name '*_test.go' | wc -l | tr -d ' ')" -lt 25
	test "$$(find . -name '*.go' -not -name '*_test.go' -print0 | xargs -0 wc -l | tail -1 | awk '{print $$1}')" -gt 2000
	test "$$(find . -name '*.go' -not -name '*_test.go' -print0 | xargs -0 wc -l | tail -1 | awk '{print $$1}')" -lt 2200
	go vet ./...
	go test -race ./...
	go build ./...

build:
	mkdir -p dist
	go build -trimpath -ldflags "-s -w -X main.version=$(VERSION)" -o dist/$(APP) ./cmd/stackbyte

package:
	VERSION=$(VERSION) sh scripts/package.sh

clean:
	rm -rf dist
