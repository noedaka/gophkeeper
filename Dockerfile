FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /server ./cmd/server/main.go
RUN go build -o /client ./cmd/client/main.go

FROM alpine:latest AS server
WORKDIR /
COPY --from=builder /server /server
EXPOSE 50051
CMD ["/server"]

FROM alpine:latest AS client
WORKDIR /
COPY --from=builder /client /client
CMD ["/client"]
