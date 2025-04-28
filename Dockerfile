# 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o news-fetcher cmd/main.go

# 运行阶段
FROM alpine:latest

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/news-fetcher .
# 复制配置文件
COPY config.yaml .

# 设置环境变量
ENV TZ=Asia/Shanghai

# 运行应用
CMD ["./news-fetcher"] 