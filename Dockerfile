# Stage 1: Build the application
FROM golang:1.24-bullseye AS builder

RUN mkdir /build

WORKDIR /app

COPY ./go.mod ./go.sum ./
RUN go mod download

COPY . ./
RUN go build -v -o /build/ ./cmd/*

# Stage 2: Copy files and configure what we need
FROM debian:bullseye-slim

RUN apt-get update && apt-get install --no-install-recommends -y ca-certificates && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /usr/local/bin/seabird-entrypoint.sh
COPY --from=builder /build /bin

RUN groupadd seabird && useradd --gid seabird seabird 
USER seabird

CMD ["/usr/local/bin/seabird-entrypoint.sh"]
