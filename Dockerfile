ARG BASE_IMAGE=alpine


FROM golang:1-bullseye AS builder

ARG VERSION=HEAD
ARG COMMIT=UNKNOWN

RUN mkdir /output

RUN apt-get update && apt-get install -y upx-ucl && apt-get clean -y && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum /usr/src/ayd/
RUN cd /usr/src/ayd && go mod download

RUN git config --global advice.detachedHead false
ARG PLUGINS="\
        ayd-mailto-alert:0.8.3 \
        ayd-slack-alert:0.8.3 \
        ayd-smb-probe:0.3.3 \
    "
RUN for x in $PLUGINS; do \
      export plugin=${x%:*} version=${x#*:} && \
      echo "download ${plugin} ${version}" && \
      git clone --depth 1 -b v${version} https://github.com/macrat/${plugin}.git /usr/src/${plugin} && \
      cd /usr/src/${plugin} && \
      go mod download; \
    done
RUN for x in $PLUGINS; do \
      export plugin=${x%:*} version=${x#*:} && \
      echo "build ${plugin} ${version}" && \
      cd /usr/src/${plugin} && \
      CGO_ENABLED=0 go build --trimpath -ldflags="-s -w -X 'main.version=${version}' -X 'main.commit=`git rev-parse --short v${version}`'" -buildvcs=false -o /output/${plugin}; \
    done

COPY . /usr/src/ayd/
RUN cd /usr/src/ayd/cmd/ayd && \
    CGO_ENABLED=0 go build --trimpath -ldflags="-s -w -X 'github.com/macrat/ayd/internal/meta.Version=$VERSION' -X 'github.com/macrat/ayd/internal/meta.Commit=$COMMIT'" -buildvcs=false -o /output/ayd

RUN upx-ucl --lzma /output/*


FROM $BASE_IMAGE

WORKDIR /var/log/ayd

COPY --from=builder /output /usr/bin
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 9000
VOLUME /var/log/ayd

ENTRYPOINT ["/usr/bin/ayd"]
