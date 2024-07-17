FROM golang:1.22-alpine AS builder
RUN apk add --no-cache git
WORKDIR /go/src/github.com/stateless-solutions

COPY . /go/src/github.com/stateless-solutions/stateless-compatibility-layer

WORKDIR /go/src/github.com/stateless-solutions/stateless-compatibility-layer
RUN CGO_ENABLED=0 GOOS=linux go build -a -o bin ./.

FROM alpine:3.20.1
WORKDIR /app
COPY --from=builder /go/src/github.com/stateless-solutions/stateless-compatibility-layer/bin ./

ENTRYPOINT ["/app/bin"]