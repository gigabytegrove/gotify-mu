FROM golang:1.26-alpine AS builder

WORKDIR /src
COPY updater/go.mod ./
COPY updater/main.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gotify-mu-updater .

FROM docker:cli

RUN apk add --no-cache ca-certificates

COPY --from=builder /out/gotify-mu-updater /usr/local/bin/gotify-mu-updater

EXPOSE 8099
ENTRYPOINT ["/usr/local/bin/gotify-mu-updater"]
