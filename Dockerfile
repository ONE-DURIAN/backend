# Build Stage
FROM golang:alpine AS builder

WORKDIR /app

# ติดตั้ง dependency ล่วงหน้าเพื่อแคชเลเยอร์
COPY go.mod go.sum ./
RUN go mod download

# ก็อปปี้โค้ดและคอมไพล์ไบนารี
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./cmd/api

# Run Stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Bangkok

WORKDIR /root/
COPY --from=builder /app/main .

EXPOSE 8080

CMD ["./main"]