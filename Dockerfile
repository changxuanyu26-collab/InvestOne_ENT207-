FROM golang:1.23-bookworm AS builder
WORKDIR /work
COPY src/ ./src/
RUN cd src && go test ./... && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o /investone .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates poppler-utils && rm -rf /var/lib/apt/lists/*
COPY --from=builder /investone /usr/local/bin/investone
ENV INVESTONE_CLOUD=1 INVESTONE_DATA_DIR=/data/users PORT=8080
EXPOSE 8080
CMD ["/usr/local/bin/investone"]
