FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o out/cloudflaere ./cmd/cloudflaere/*.go


FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root
COPY --from=builder /app/out/ /usr/local/bin/

ENTRYPOINT ["cloudflaere"]
