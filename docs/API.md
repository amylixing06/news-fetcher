# API 文档

## 新闻抓取 API

### 获取新闻列表

**请求**
```
GET /api/v1/news
```

**参数**
- `page`: 页码，默认 1
- `size`: 每页数量，默认 10
- `source`: 新闻源，可选

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total": 100,
    "list": [
      {
        "id": "123",
        "title": "新闻标题",
        "content": "新闻内容",
        "source": "新闻源",
        "url": "新闻链接",
        "publish_time": "2024-04-28T12:00:00Z",
        "analysis": "AI 分析结果"
      }
    ]
  }
}
```

## 配置 API

### 获取配置

**请求**
```
GET /api/v1/config
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "sources": [
      {
        "name": "新闻源名称",
        "url": "API 地址",
        "enabled": true
      }
    ],
    "telegram": {
      "bot_token": "机器人 Token",
      "chat_id": "聊天 ID"
    },
    "ai": {
      "enabled": true,
      "api_key": "API Key",
      "model": "模型名称"
    },
    "cache": {
      "ttl": 3600
    }
  }
}
```

### 更新配置

**请求**
```
PUT /api/v1/config
```

**请求体**
```json
{
  "sources": [
    {
      "name": "新闻源名称",
      "url": "API 地址",
      "enabled": true
    }
  ],
  "telegram": {
    "bot_token": "机器人 Token",
    "chat_id": "聊天 ID"
  },
  "ai": {
    "enabled": true,
    "api_key": "API Key",
    "model": "模型名称"
  },
  "cache": {
    "ttl": 3600
  }
}
```

**响应**
```json
{
  "code": 0,
  "message": "success"
}
```

## 状态 API

### 获取服务状态

**请求**
```
GET /api/v1/status
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "running",
    "uptime": "1h30m",
    "last_fetch": "2024-04-28T12:00:00Z",
    "news_count": 100,
    "error_count": 0
  }
}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 1001 | 参数错误 |
| 1002 | 配置错误 |
| 1003 | 服务错误 |
| 1004 | 网络错误 |
| 1005 | 认证错误 | 