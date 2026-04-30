# Stage 1: Build
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

# Install templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN templ generate
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /kvt ./cmd/kvt

# Stage 2: Runtime
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /kvt /usr/local/bin/kvt

# Data directory for registry and project databases
RUN mkdir -p /data/.tasks
ENV HOME=/data

EXPOSE 7842

ENTRYPOINT ["kvt"]
CMD ["serve", "--host", "0.0.0.0", "--port", "7842"]
