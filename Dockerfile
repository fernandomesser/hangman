FROM golang:1.21-alpine as builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o hangman .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/hangman .
COPY static ./static
COPY templates ./templates
CMD ["./hangman"]