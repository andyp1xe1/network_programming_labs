FROM golang:1.25-alpine

WORKDIR /app

RUN --mount=type=bind,source=.,target=/src \
    cd /src && go build -o /app/kvstore ./cmd/kvstore

EXPOSE 8000 9000

CMD ["./kvstore"]
