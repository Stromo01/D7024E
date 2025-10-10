FROM golang:1.20 AS builder
WORKDIR /src

# copy go.mod first for caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# build the main in ./cmd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /bin/kademlia ./cmd

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/kademlia /usr/local/bin/kademlia
ENTRYPOINT ["/usr/local/bin/kademlia"]