FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/apiserver

FROM alpine:3.22

COPY --from=builder /app/server /app/server

USER 10001:10001

EXPOSE 8080

CMD ["/app/server"]