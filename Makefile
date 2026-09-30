BINARY   := cfgtool
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
REPORTS  := reports

.PHONY: build test vet fmt check sec vuln staticcheck release clean

build:
	go build -o bin/$(BINARY) ./cmd/cfgtool

test:
	go test ./... -count=1

vet:
	go vet ./...

fmt:
	gofmt -l .

check: fmt vet test

sec:
	mkdir -p $(REPORTS)
	gosec -fmt=text -out=$(REPORTS)/gosec.txt -no-fail ./...
	gosec -fmt=text ./... || true

vuln:
	mkdir -p $(REPORTS)
	govulncheck ./... > $(REPORTS)/govulncheck.txt 2>&1 || true
	cat $(REPORTS)/govulncheck.txt

staticcheck:
	mkdir -p $(REPORTS)
	staticcheck ./... > $(REPORTS)/staticcheck.txt 2>&1 || true
	cat $(REPORTS)/staticcheck.txt

release:
	./scripts/release.sh $(VERSION)

clean:
	rm -rf bin dist
