GOOS=darwin
GOARCH=amd64
DIST_BUILD=haruno

build:
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o build/$(DIST_BUILD)

debug:
	go run haruno.go

.PHONY: build debug


