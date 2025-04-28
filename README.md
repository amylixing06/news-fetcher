# 资讯聚合推送服务

一个基于 Go 语言的资讯聚合和推送服务，可以从多个数据源获取信息，进行翻译、AI 分析后推送到 Telegram。

## 主要功能

1. **多数据源抓取**：支持 API、RSS 等多种数据源
2. **内容翻译**：英文内容自动翻译成中文
3. **AI 分析**：分析市场情绪和山寨季指数
4. **消息推送**：通过 Telegram Bot 推送到群聊
5. **定时抓取**：定时自动获取最新资讯
6. **去重处理**：避免重复推送相同内容

## 环境要求

- Go 1.20 或更高版本
- Docker 和 Docker Compose (用于容器化部署)

## 快速开始

### 1. 克隆仓库

```bash
git clone <repository-url>
cd news-fetcher
```

### 2. 配置

修改 `config.yaml` 文件，设置数据源和 Telegram Bot 配置：

```yaml
api_sources:
  - url: https://api.theblockbeats.news/v1/open-api/open-flash?page=1&size=1&type=push&lang=cn
    retry_count: 3
    retry_interval: 5
    timeout: 30
telegram:
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  chat_ids:
    - "YOUR_CHAT_ID"
  enabled: true
  retry_count: 3
  retry_interval: 5
  proxy_url: "http://127.0.0.1:7890"  # 可选，使用HTTP代理
  timeout: 60
  max_msg_length: 4096
fetch_interval: 300  # 抓取间隔(秒)
cache:
  type: memory
  expiration: 3600
ai:
  model: "deepseek"
  max_tokens: 1000
  temperature: 0.7
  timeout: 30
```

### 3. 运行服务

```bash
go run cmd/main.go
```

### 4. 测试推送

```bash
go run test_telegram.go
```

## 配置说明

### 配置文件 (config.yaml)

```yaml
api_sources:
  - url: https://api.theblockbeats.news/v1/open-api/open-flash?page=1&size=1&type=push&lang=cn
    retry_count: 3
    retry_interval: 5
    timeout: 30
telegram:
  token: "YOUR_TELEGRAM_BOT_TOKEN"
  chat_ids:
    - "YOUR_CHAT_ID"
  enabled: true
    retry_count: 3
    retry_interval: 5
  proxy_url: "http://127.0.0.1:7890"  # 可选，使用HTTP代理
  timeout: 60
  max_msg_length: 4096
fetch_interval: 300  # 抓取间隔(秒)
cache:
  type: memory
  expiration: 3600
ai:
  model: "deepseek"
  max_tokens: 1000
  temperature: 0.7
  timeout: 30
```

## 项目结构

```
news-fetcher/
├── cmd/
│   ├── main.go         # 程序入口
│   └── test_telegram.go # Telegram测试程序
├── internal/
│   ├── ai/             # AI 分析模块
│   ├── cache/          # 缓存模块
│   ├── config/         # 配置管理
│   ├── db/             # 数据库操作
│   ├── fetcher/        # 数据源抓取
│   ├── models/         # 数据模型
│   ├── sender/         # 消息发送
│   └── translator/     # 翻译模块
├── data/               # 数据存储目录
├── config.yaml         # 配置文件
└── README.md           # 项目说明
```

## 后续扩展

1. 新增更多数据源支持（如 OKX、Binance 等交易所 API）
2. 增加更多消息推送渠道（如飞书、钉钉等）
3. 增强 AI 分析能力，提供更精准的市场预测
4. 添加 Web 管理界面，方便配置和查看历史消息