FROM golang:1.24-alpine AS builder

WORKDIR /app

ENV GOTOOLCHAIN=auto

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o namecheap-webhook ./cmd/webhook

FROM alpine:3.20
RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/namecheap-webhook .

EXPOSE 8888 8080

CMD ["./namecheap-webhook"]