FROM golang:1.26-alpine AS builder

WORKDIR /src
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/movietracker-server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H appuser

WORKDIR /app
COPY --from=builder /out/movietracker-server /app/movietracker-server
COPY migrations/server /app/migrations/server

ENV ADDR=:8080
ENV DB_PATH=/data/server.db

EXPOSE 8080

USER appuser
ENTRYPOINT ["/app/movietracker-server"]
