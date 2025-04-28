package test

import (
	"context"
	"testing"
	"time"

	"github.com/amylixing/news-fetcher/internal/cache"
	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/models"
	"github.com/amylixing/news-fetcher/internal/sender"
	"github.com/stretchr/testify/assert"
)

func TestSender(t *testing.T) {
	// 加载测试配置
	cfg, err := config.LoadConfig("../config/config.yaml")
	assert.NoError(t, err)

	// 创建缓存
	newsCache, err := cache.NewCache("test_cache.json")
	assert.NoError(t, err)

	// 创建发送器
	s, err := sender.NewSender(cfg.Telegram, newsCache)
	assert.NoError(t, err)

	// 创建测试新闻
	news := &models.News{
		ID:              "test-123",
		OriginalTitle:   "测试新闻标题",
		OriginalContent: "这是一条测试新闻内容",
		Source:          "test",
		Link:            "https://example.com/test",
		CreateTime:      time.Now(),
		Analysis:        "这是一条测试新闻的AI分析结果",
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试发送
	err = s.SendNews(ctx, news)
	assert.NoError(t, err)
}
