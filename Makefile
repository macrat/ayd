SOURCES = $(shell find . -iname '*.go')
VERSION = $(shell git describe --tags --dirty | grep -o '[0-9].*')
COMMIT = $(shell git rev-parse --short $(shell git describe))


ayd: ${SOURCES}
	CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/macrat/ayd/internal/meta.Version=${VERSION} -X github.com/macrat/ayd/internal/meta.Commit=${COMMIT}" -trimpath -o ayd ./cmd/ayd


.PHONY: test containertest cover fmt resources clean install

test:
	go test -race -cover ./...

containertest:
	cd testdata && docker compose up -d
	go test -race -cover -tags=container ./...; cd testdata && docker compose down -v

cover:
	go test -race -coverprofile=cov ./... && go tool cover -html=cov; rm cov

fmt:
	gofmt -s -w ${SOURCES}

resources: cmd/ayd/resource_windows_386.syso cmd/ayd/resource_windows_amd64.syso cmd/ayd/resource_windows_arm.syso cmd/ayd/resource_windows_arm64.syso

%.syso: cmd/ayd/versioninfo.json
	cd cmd/ayd && go run github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest -platform-specific

clean:
	-rm ayd ayd_*.log

install: ayd
	sudo install ./ayd /usr/local/bin/ayd
