package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CacheItem 缓存项
type CacheItem struct {
	Value     interface{} `json:"value"`
	Source    string      `json:"source"`
	Timestamp time.Time   `json:"timestamp"`
	ExpireAt  time.Time   `json:"expire_at"`
}

// Cache 缓存结构
type Cache struct {
	items    map[string]CacheItem
	filePath string
	mu       sync.RWMutex
}

// Clear 清空缓存
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]CacheItem)
	log.Printf("[缓存] 已清空所有缓存项")

	// 保存空缓存到文件
	return c.save()
}

// NewCache 创建新的缓存实例
func NewCache(filePath string) (*Cache, error) {
	// 确保缓存目录存在
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, fmt.Errorf("创建缓存目录失败: %v", err)
	}

	// 创建新的空缓存实例
	cache := &Cache{
		items:    make(map[string]CacheItem),
		filePath: filePath,
	}

	// 清空缓存
	if err := cache.Clear(); err != nil {
		return nil, fmt.Errorf("清空缓存失败: %v", err)
	}

	// 启动定期清理
	go cache.startCleanup()

	log.Printf("缓存系统初始化完成，文件路径: %s", filePath)
	return cache, nil
}

// generateKey 生成缓存键
func generateKey(source string, id interface{}) string {
	// 直接使用 source 和 id 的组合作为键，不使用 MD5 哈希
	return fmt.Sprintf("%s:%v", source, id)
}

// Set 设置缓存
func (c *Cache) Set(source string, id interface{}, value interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := generateKey(source, id)
	now := time.Now()

	// 确保 TTL 不超过一年
	if ttl > 365*24*time.Hour {
		ttl = 365 * 24 * time.Hour
		log.Printf("[缓存] TTL 超过一年，已调整为一年: [%s] ID: %v", source, id)
	}

	expireAt := now.Add(ttl)

	c.items[key] = CacheItem{
		Value:     value,
		Source:    source,
		Timestamp: now,
		ExpireAt:  expireAt,
	}

	log.Printf("[缓存] 设置缓存项: [%s] ID: %v, 过期时间: %v", source, id, expireAt.Format("2006-01-02 15:04:05"))

	// 保存到文件
	return c.save()
}

// save 保存缓存到文件（内部方法，调用前需要持有锁）
func (c *Cache) save() error {
	data, err := json.MarshalIndent(c.items, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化缓存数据失败: %v", err)
	}

	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		return fmt.Errorf("写入缓存文件失败: %v", err)
	}

	log.Printf("[缓存] 缓存已保存到文件: %s", c.filePath)
	return nil
}

// Save 保存缓存到文件（公开方法）
func (c *Cache) Save() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.save()
}

// Get 获取缓存
func (c *Cache) Get(source string, id interface{}) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := generateKey(source, id)
	item, exists := c.items[key]
	if !exists {
		log.Printf("[缓存] 缓存未命中: [%s] ID: %v", source, id)
		return nil, false
	}

	now := time.Now()
	// 检查是否过期
	if now.After(item.ExpireAt) {
		log.Printf("[缓存] 缓存项已过期: [%s] ID: %v, 过期时间: %v", source, id, item.ExpireAt.Format("2006-01-02 15:04:05"))
		go c.Delete(source, id) // 异步删除过期项
		return nil, false
	}

	log.Printf("[缓存] 缓存命中: [%s] ID: %v, 设置时间: %v", source, id, item.Timestamp.Format("2006-01-02 15:04:05"))
	return item.Value, true
}

// Delete 删除缓存
func (c *Cache) Delete(source string, id interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := generateKey(source, id)
	delete(c.items, key)
	log.Printf("[缓存] 删除缓存项: [%s] ID: %v", source, id)

	// 立即保存到文件
	if err := c.save(); err != nil {
		log.Printf("[缓存] 保存缓存文件失败: %v", err)
		return err
	}
	return nil
}

// Load 从文件加载缓存
func (c *Cache) Load() error {
	data, err := os.ReadFile(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return err // 文件不存在，让调用者处理
		}
		return fmt.Errorf("读取缓存文件失败: %v", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := json.Unmarshal(data, &c.items); err != nil {
		return fmt.Errorf("解析缓存数据失败: %v", err)
	}

	// 验证并清理无效的时间戳
	now := time.Now()
	for key, item := range c.items {
		// 如果时间戳在未来，或者过期时间在未来超过1年，认为这是无效的
		if item.Timestamp.After(now) || item.ExpireAt.After(now.Add(365*24*time.Hour)) {
			log.Printf("[缓存] 发现无效的时间戳，删除缓存项: [%s] ID: %v", item.Source, key)
			delete(c.items, key)
		}
	}

	log.Printf("从文件加载缓存成功，共 %d 条记录", len(c.items))
	return nil
}

// startCleanup 启动定期清理
func (c *Cache) startCleanup() {
	ticker := time.NewTicker(5 * time.Minute) // 每5分钟清理一次
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup 清理过期的缓存项
func (c *Cache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	cleaned := 0
	for key, item := range c.items {
		if now.After(item.ExpireAt) {
			delete(c.items, key)
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Printf("[缓存] 清理了 %d 个过期缓存项", cleaned)
		c.save() // 保存清理后的缓存
	}
}
