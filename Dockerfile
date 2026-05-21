FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy everything first — go.sum may be stale, will tidy below
COPY go.mod go.sum ./
# go mod download fails if go.sum is stale, so run tidy first
RUN go mod tidy && go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /api ./cmd/api/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl
COPY --from=builder /api /usr/local/bin/api
COPY migrations/ /migrations/

EXPOSE 8080
ENTRYPOINT ["api"]
