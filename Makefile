.PHONY: build test clean run

# 默认目标
all: build

# 构建
build:
	@echo "构建项目..."
	go build -o bin/news-fetcher cmd/main.go

# 运行测试
test:
	@echo "运行测试..."
	go test -v ./test/...

# 清理
clean:
	@echo "清理构建文件..."
	rm -rf bin/
	go clean

# 运行
run: build
	@echo "运行服务..."
	./bin/news-fetcher

# 安装依赖
deps:
	@echo "安装依赖..."
	go mod tidy

# 格式化代码
fmt:
	@echo "格式化代码..."
	go fmt ./...

# 检查代码
vet:
	@echo "检查代码..."
	go vet ./...

# 代码质量检查
lint:
	@echo "运行代码质量检查..."
	golangci-lint run

# 构建 Docker 镜像
docker-build:
	@echo "构建 Docker 镜像..."
	docker build -t news-fetcher .

# 运行 Docker 容器
docker-run:
	@echo "运行 Docker 容器..."
	docker run -d --name news-fetcher news-fetcher

# 停止 Docker 容器
docker-stop:
	@echo "停止 Docker 容器..."
	docker stop news-fetcher
	docker rm news-fetcher 