ARG BASE_IMAGE


FROM golang:latest AS builder

ARG VERSION="head on container"
ARG COMMIT=UNKNOWN

ENV CGO_ENABLED 0

RUN mkdir /output

WORKDIR /usr/src/ayd
COPY ayd .
RUN go build -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -o /output/ayd

WORKDIR /usr/src/ayd-mail-alert
COPY ayd-mail-alert .
RUN go build -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -o /output/ayd-mail-alert

WORKDIR /usr/src/ayd-slack-alert
COPY ayd-slack-alert .
RUN go build -ldflags="-s -w -X 'main.version=$VERSION' -X 'main.commit=$COMMIT'" -o /output/ayd-slack-alert


FROM $BASE_IMAGE

WORKDIR /var/log/ayd

COPY --from=builder /output /usr/bin
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 9000
VOLUME /var/log/ayd

ENTRYPOINT ["ayd"]
