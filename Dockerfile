FROM golang:alpine AS builder
WORKDIR /app
COPY / ./
RUN apk add build-base && apk cache clean
ENV CGO_ENABLED=1
RUN go build -o ./recruit ./main.go


FROM alpine
WORKDIR /app
COPY --from=builder /app/recruit ./recruit
EXPOSE 8000
CMD ["./recruit"]