package test

import (
	"context"
	"testing"
	"time"

	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/fetcher"
	"github.com/stretchr/testify/assert"
)

func TestFetcher(t *testing.T) {
	// 加载测试配置
	cfg, err := config.LoadConfig("../config/config.yaml")
	assert.NoError(t, err)

	// 创建抓取器
	f, err := fetcher.NewFetcher(cfg.Sources)
	assert.NoError(t, err)

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试抓取
	newsList, err := f.Fetch(ctx)
	assert.NoError(t, err)
	assert.NotEmpty(t, newsList)

	// 验证新闻内容
	for _, news := range newsList {
		assert.NotEmpty(t, news.ID)
		assert.NotEmpty(t, news.OriginalTitle)
		assert.NotEmpty(t, news.OriginalContent)
		assert.NotEmpty(t, news.Source)
		assert.NotEmpty(t, news.Link)
		assert.NotZero(t, news.CreateTime)
	}
}
