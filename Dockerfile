# Stage 1: Build the application
FROM golang:1.14-buster as builder

RUN mkdir /build && mkdir /seabird-stock-plugin

WORKDIR /seabird-stock-plugin
ADD ./go.mod ./go.sum ./
RUN go mod download

ADD . ./

RUN go build -v -o /build/seabird-stock-plugin ./cmd/seabird-stock-plugin

# Stage 2: Copy files and configure what we need
FROM debian:buster-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Copy the built seabird into the container
COPY --from=builder /build/seabird-stock-plugin /usr/local/bin

ENTRYPOINT ["/usr/local/bin/seabird-stock-plugin"]