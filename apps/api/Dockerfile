FROM docker.io/library/golang:1.26-alpine AS builder

WORKDIR /src/apps/api

RUN apk add --no-cache git

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/worker ./cmd/worker && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM docker.io/library/alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -S app && \
    adduser -S -D -H -h /app -s /sbin/nologin -G app app

WORKDIR /app

COPY --from=builder /out/server /app/server
COPY --from=builder /out/worker /app/worker
COPY --from=builder /out/scheduler /app/scheduler
COPY --from=builder /out/migrate /app/migrate
COPY apps/api/migrations /app/migrations

USER app

ENV HTTP_PORT=8080

EXPOSE 8080

ENTRYPOINT ["/app/server"]
