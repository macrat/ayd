SOURCES = $(shell find . -iname '*.go')
VERSION = $(shell git describe --tags --dirty | grep -o '[0-9].*')
COMMIT = $(shell git rev-parse --short $(shell git describe))


ayd: ${SOURCES}
	go build -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" -trimpath .


.PHONY: test cover fmt clean install

test:
	go test -race -cover ./...

cover:
	go test -race -coverprofile=cov ./... && go tool cover -html=cov; rm cov

fmt:
	gofmt -s -w ${SOURCES}

clean:
	-rm ayd

install: ayd
	sudo install ./ayd /usr/local/bin/ayd
