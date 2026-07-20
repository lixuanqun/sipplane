# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/sipplane ./cmd/sipplane \
 && CGO_ENABLED=0 go build -o /out/sipplane-control ./cmd/sipplane-control \
 && CGO_ENABLED=0 go build -o /out/sipplanectl ./cmd/sipplanectl

FROM alpine:3.20
RUN apk add --no-cache ca-certificates \
 && adduser -D -H -u 65532 sipplane
COPY --from=build /out/sipplane /usr/local/bin/sipplane
COPY --from=build /out/sipplane-control /usr/local/bin/sipplane-control
COPY --from=build /out/sipplanectl /usr/local/bin/sipplanectl
COPY --chown=65532:65532 examples/config /etc/sipplane
USER 65532:65532
EXPOSE 5060/udp 5060/tcp 8080 8090
ENTRYPOINT ["sipplane", "-config", "/etc/sipplane/bootstrap.yaml", "-resources", "/etc/sipplane"]
