package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/models"
	"github.com/mmcdole/gofeed"
)

// Fetcher 新闻抓取器
type Fetcher struct {
	apiSources []*APISource
	rssSources []*RSSSource
	client     *http.Client
}

// Source 数据源接口
type Source interface {
	Fetch(ctx context.Context) ([]*models.News, error)
}

// APISource API数据源
type APISource struct {
	config     *config.SourceConfig
	httpClient *http.Client
}

// RSSSource RSS数据源
type RSSSource struct {
	config     *config.SourceConfig
	proxyURL   string
	httpClient *http.Client
}

// NewFetcher 创建新闻抓取器
func NewFetcher(cfg *config.SourcesConfig) (*Fetcher, error) {
	// 创建HTTP客户端
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	// 初始化API数据源
	var apiSources []*APISource
	for _, apiCfg := range cfg.API {
		source := NewAPISource(apiCfg, client)
		apiSources = append(apiSources, source)
	}

	// 初始化RSS数据源
	var rssSources []*RSSSource
	for _, rssCfg := range cfg.RSS {
		source := NewRSSSource(rssCfg)
		rssSources = append(rssSources, source)
	}

	return &Fetcher{
		apiSources: apiSources,
		rssSources: rssSources,
		client:     client,
	}, nil
}

// NewAPISource 创建API数据源
func NewAPISource(cfg *config.SourceConfig, client *http.Client) *APISource {
	return &APISource{
		config:     cfg,
		httpClient: client,
	}
}

// NewRSSSource 创建RSS数据源
func NewRSSSource(cfg *config.SourceConfig) *RSSSource {
	// 创建 HTTP 客户端
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     30 * time.Second,
	}

	// 配置代理
	if cfg.ProxyURL != "" {
		if proxyURL, err := url.Parse(cfg.ProxyURL); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
			log.Printf("已配置代理: %s", cfg.ProxyURL)
		} else {
			log.Printf("解析代理 URL 失败: %v", err)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	return &RSSSource{
		config:     cfg,
		proxyURL:   cfg.ProxyURL,
		httpClient: client,
	}
}

// Fetch 从所有数据源抓取新闻
func (f *Fetcher) Fetch(ctx context.Context) ([]*models.News, error) {
	var allNews []*models.News

	// 从API源抓取
	for _, source := range f.apiSources {
		log.Printf("开始从 %s 获取新闻...", source.config.URL)
		news, err := source.Fetch(ctx)
		if err != nil {
			log.Printf("从 %s 获取新闻失败: %v", source.config.URL, err)
			continue
		}
		log.Printf("从 %s 获取到 %d 条新闻", source.config.URL, len(news))
		allNews = append(allNews, news...)
	}

	// 从RSS源抓取
	for _, source := range f.rssSources {
		news, err := source.Fetch(ctx)
		if err != nil {
			log.Printf("从RSS源抓取新闻失败: %v", err)
			continue
		}
		allNews = append(allNews, news...)
	}

	if len(allNews) == 0 {
		log.Println("未获取到任何新闻")
		return nil, nil
	}

	log.Printf("本次共获取到 %d 条新闻", len(allNews))
	return allNews, nil
}

// Fetch 从API数据源抓取新闻
func (s *APISource) Fetch(ctx context.Context) ([]*models.News, error) {
	log.Printf("开始请求API: %s", s.config.URL)

	// 添加请求参数
	reqURL := s.config.URL
	if len(s.config.Params) > 0 {
		values := url.Values{}
		for k, v := range s.config.Params {
			values.Add(k, fmt.Sprintf("%v", v))
		}
		if strings.Contains(reqURL, "?") {
			reqURL += "&" + values.Encode()
		} else {
			reqURL += "?" + values.Encode()
		}
		log.Printf("请求URL（带参数）: %s", reqURL)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加请求头
	for key, value := range s.config.Headers {
		req.Header.Add(key, value)
		log.Printf("添加请求头: %s = %s", key, value)
	}

	// 发送请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API响应状态码异常: %d", resp.StatusCode)
	}

	log.Printf("API请求成功")

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 打印响应内容
	log.Printf("API响应内容: %s", string(body))

	// 解析响应
	var response models.APIResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if response.Status != 0 {
		return nil, fmt.Errorf("API返回错误: %s", response.Message)
	}

	// 解析新闻列表
	var newsList []*models.News
	for _, item := range response.Data.List {
		news := &models.News{
			ID:              item.ID,
			OriginalTitle:   item.Title,
			OriginalContent: item.Content,
			Source:          s.config.URL,
			CreateTime:      time.Now(),
		}
		newsList = append(newsList, news)
		log.Printf("解析新闻: %s (ID: %v)", news.OriginalTitle, news.ID)
	}

	log.Printf("成功解析 %d 条新闻", len(newsList))
	return newsList, nil
}

// Fetch 从RSS数据源抓取新闻
func (s *RSSSource) Fetch(ctx context.Context) ([]*models.News, error) {
	log.Printf("开始抓取 RSS 源: %s", s.config.URL)

	var feed *gofeed.Feed
	retryCount := 0
	maxRetries := s.config.Retry.Count

	for retryCount <= maxRetries {
		// 发送请求
		req, err := http.NewRequestWithContext(ctx, "GET", s.config.URL, nil)
		if err != nil {
			log.Printf("创建请求失败: %v", err)
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			retryCount++
			if retryCount <= maxRetries {
				log.Printf("发送请求失败，正在重试 (%d/%d): %v", retryCount, maxRetries, err)
				time.Sleep(time.Duration(s.config.Retry.Interval) * time.Second)
				continue
			}
			log.Printf("发送请求失败，已达到最大重试次数: %v", err)
			return nil, fmt.Errorf("发送请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			retryCount++
			if retryCount <= maxRetries {
				log.Printf("请求失败，状态码: %d，正在重试 (%d/%d)", resp.StatusCode, retryCount, maxRetries)
				time.Sleep(time.Duration(s.config.Retry.Interval) * time.Second)
				continue
			}
			log.Printf("请求失败，状态码: %d，已达到最大重试次数", resp.StatusCode)
			return nil, fmt.Errorf("请求失败，状态码: %d", resp.StatusCode)
		}

		// 解析 RSS
		fp := gofeed.NewParser()
		feed, err = fp.Parse(resp.Body)
		if err != nil {
			retryCount++
			if retryCount <= maxRetries {
				log.Printf("解析 RSS 失败，正在重试 (%d/%d): %v", retryCount, maxRetries, err)
				time.Sleep(time.Duration(s.config.Retry.Interval) * time.Second)
				continue
			}
			log.Printf("解析 RSS 失败，已达到最大重试次数: %v", err)
			return nil, fmt.Errorf("解析 RSS 失败: %v", err)
		}

		break
	}

	if feed == nil {
		return nil, fmt.Errorf("获取 RSS 源失败")
	}

	// 转换新闻
	var newsList []*models.News
	for _, item := range feed.Items {
		// 如果没有 GUID，使用 Link 作为 ID
		id := item.GUID
		if id == "" {
			id = item.Link
		}

		// 如果发布时间为空，使用当前时间
		createTime := time.Now()
		if item.PublishedParsed != nil {
			createTime = *item.PublishedParsed
		}

		news := &models.News{
			ID:              id,
			OriginalTitle:   item.Title,
			OriginalContent: item.Description,
			Link:            item.Link,
			Source:          s.config.URL,
			CreateTime:      createTime,
		}
		newsList = append(newsList, news)
		log.Printf("解析RSS新闻: %s (ID: %v)", news.OriginalTitle, news.ID)
	}

	log.Printf("成功抓取 %d 条新闻", len(newsList))
	return newsList, nil
}
