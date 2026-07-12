FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /ledger .

FROM alpine:latest
COPY --from=builder /ledger /ledger

EXPOSE 8003
CMD ["/ledger"]
