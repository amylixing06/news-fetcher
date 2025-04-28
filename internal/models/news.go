package models

import (
	"sync"
	"time"
)

// News 表示一条新闻数据
type News struct {
	ID                string    `json:"id"`
	OriginalTitle     string    `json:"original_title"`
	OriginalContent   string    `json:"original_content"`
	TranslatedTitle   string    `json:"translated_title"`
	TranslatedContent string    `json:"translated_content"`
	Analysis          string    `json:"analysis"` // AI分析结果
	Link              string    `json:"link"`
	CreateTime        time.Time `json:"create_time"`
	Source            string    `json:"source"`
}

// APIResponse API响应
type APIResponse struct {
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Data    APIResponseData `json:"data"`
}

// APIResponseData API响应数据
type APIResponseData struct {
	List []APIResponseItem `json:"list"`
}

// APIResponseItem API响应项
type APIResponseItem struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

// NewsCache 新闻缓存
type NewsCache struct {
	sync.RWMutex
	items map[string]time.Time
	ttl   time.Duration
}

// NewNewsCache 创建新闻缓存
func NewNewsCache(ttl time.Duration) *NewsCache {
	return &NewsCache{
		items: make(map[string]time.Time),
		ttl:   ttl,
	}
}

// Add 添加新闻到缓存
func (c *NewsCache) Add(id string) {
	c.Lock()
	defer c.Unlock()
	c.items[id] = time.Now()
}

// Exists 检查新闻是否存在于缓存中
func (c *NewsCache) Exists(id string) bool {
	c.RLock()
	defer c.RUnlock()

	timestamp, exists := c.items[id]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Since(timestamp) > c.ttl {
		delete(c.items, id)
		return false
	}

	return true
}

// Cleanup 清理过期的缓存项
func (c *NewsCache) Cleanup() {
	c.Lock()
	defer c.Unlock()

	now := time.Now()
	for id, timestamp := range c.items {
		if now.Sub(timestamp) > c.ttl {
			delete(c.items, id)
		}
	}
}
