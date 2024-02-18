FROM golang:alpine AS builder
WORKDIR /app
COPY / ./
RUN apk --no-cache add build-base
ENV CGO_ENABLED=1
RUN go build -o ./recruit ./main.go
CMD ["./recruit"]


# FROM alpine
# WORKDIR /app
# COPY --from=builder /app/recruit ./recruit
# EXPOSE 8000