FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o targetservice ./main.go

# final slim image
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/targetservice .
COPY .env .

EXPOSE 2112

CMD ["./targetservice"]
