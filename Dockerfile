FROM golang:1.23-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /golinks ./cmd/golinks

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /golinks /usr/local/bin/golinks
EXPOSE 8080
USER 65534:65534
ENTRYPOINT ["golinks"]
