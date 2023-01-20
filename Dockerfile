ARG BASE_IMAGE=alpine


FROM golang:latest AS builder

ARG VERSION=HEAD
ARG COMMIT=UNKNOWN

ENV CGO_ENABLED 0

RUN mkdir /output

COPY . /usr/src/ayd

RUN cd /usr/src/ayd/cmd/ayd && go build --trimpath -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -buildvcs=false -o /output/ayd

RUN for x in \
      ayd-mailto-alert:0.8.0 \
      ayd-slack-alert:0.8.0  \
      ayd-smb-probe:0.3.1    \
    ; do \
      export plugin=${x%:*} version=${x#*:} && \
      git clone --depth 1 -b v${version} https://github.com/macrat/${plugin}.git /usr/src/${plugin} && \
      cd /usr/src/${plugin} && \
      go build --trimpath -ldflags="-s -w -X 'main.version=${version}' -X 'main.commit=`git rev-parse --short v${version}`'" -buildvcs=false -o /output/${plugin}; \
    done

RUN apt-get update && apt-get install -y upx && upx --lzma /output/*


FROM $BASE_IMAGE

WORKDIR /var/log/ayd

COPY --from=builder /output /usr/bin
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 9000
VOLUME /var/log/ayd

ENTRYPOINT ["/usr/bin/ayd"]
