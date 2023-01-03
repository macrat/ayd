ARG BASE_IMAGE=alpine


FROM golang:latest AS builder

ARG VERSION=HEAD
ARG COMMIT=UNKNOWN

ENV CGO_ENABLED 0

RUN mkdir /output

COPY . /usr/src

RUN cd /usr/src/cmd/ayd && go build --trimpath -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -buildvcs=false -o /output/ayd
RUN cd /usr/src/_plugins/mailto-alert && go build --trimpath -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -buildvcs=false -o /output/ayd-mailto-alert
RUN cd /usr/src/_plugins/slack-alert && go build --trimpath -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -buildvcs=false -o /output/ayd-slack-alert
RUN cd /usr/src/_plugins/smb-probe && go build --trimpath -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -buildvcs=false -o /output/ayd-smb-probe

RUN apt-get update && apt-get install -y upx && upx --lzma /output/*


FROM $BASE_IMAGE

WORKDIR /var/log/ayd

COPY --from=builder /output /usr/bin
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 9000
VOLUME /var/log/ayd

ENTRYPOINT ["/usr/bin/ayd"]
