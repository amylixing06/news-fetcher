package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/amylixing/news-fetcher/internal/ai"
	"github.com/amylixing/news-fetcher/internal/cache"
	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/fetcher"
	"github.com/amylixing/news-fetcher/internal/models"
	"github.com/amylixing/news-fetcher/internal/sender"
)

type App struct {
	cfg      *config.Config
	fetcher  *fetcher.Fetcher
	analyzer *ai.Analyzer
	sender   *sender.Sender
	cache    *cache.Cache
}

func NewApp(cfg *config.Config) (*App, error) {
	// 创建数据目录
	dataDir := filepath.Join(os.Getenv("HOME"), ".news-fetcher")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %v", err)
	}
	log.Printf("数据目录已创建: %s", dataDir)

	// 初始化缓存
	cacheFile := filepath.Join(dataDir, "news_cache.json")
	log.Printf("初始化缓存，文件路径: %s", cacheFile)
	newsCache, err := cache.NewCache(cacheFile)
	if err != nil {
		return nil, fmt.Errorf("初始化缓存失败: %v", err)
	}
	log.Printf("缓存初始化成功")

	// 初始化抓取器
	log.Printf("初始化抓取器...")
	fetcher, err := fetcher.NewFetcher(cfg.Sources)
	if err != nil {
		return nil, fmt.Errorf("初始化抓取器失败: %v", err)
	}
	log.Printf("抓取器初始化成功")

	// 初始化AI分析器
	log.Printf("初始化AI分析器...")
	analyzer, err := ai.NewAnalyzer(cfg.AI)
	if err != nil {
		log.Printf("AI分析器初始化失败: %v", err)
		cfg.AI.Enabled = false
	} else {
		log.Printf("AI分析器初始化成功")
	}

	// 初始化发送器
	log.Printf("初始化发送器...")
	sender, err := sender.NewSender(cfg.Telegram, newsCache)
	if err != nil {
		return nil, fmt.Errorf("初始化发送器失败: %v", err)
	}
	log.Printf("发送器初始化成功")

	app := &App{
		cfg:      cfg,
		fetcher:  fetcher,
		analyzer: analyzer,
		sender:   sender,
		cache:    newsCache,
	}

	log.Printf("应用初始化完成")
	return app, nil
}

func (app *App) Run(ctx context.Context) error {
	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 创建等待组
	var wg sync.WaitGroup
	
	// 启动定时任务
	ticker := time.NewTicker(time.Duration(app.cfg.App.FetchInterval) * time.Second)
	defer ticker.Stop()

	// 启动主循环
		wg.Add(1)
	go func() {
			defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				log.Println("收到停止信号，正在退出...")
				return
			case <-ticker.C:
				if err := app.processNews(ctx); err != nil {
					log.Printf("处理新闻失败: %v", err)
				}
			}
			}
	}()

	// 等待信号
	<-sigChan
	log.Println("收到终止信号，正在关闭服务...")
	wg.Wait()
	return nil
}

func (a *App) processNews(ctx context.Context) error {
	// 获取新闻列表
	newsList, err := a.fetcher.Fetch(ctx)
			if err != nil {
		return fmt.Errorf("获取新闻失败: %v", err)
	}

	log.Printf("原始获取到 %d 条新闻", len(newsList))

	// 如果没有新闻，直接返回
	if len(newsList) == 0 {
		log.Printf("没有获取到新的新闻")
		return nil
	}
	
	// 过滤并处理新新闻
	var newNews []*models.News
	for _, news := range newsList {
		// 检查缓存
		if _, exists := a.cache.Get(news.Source, news.ID); !exists {
			log.Printf("发现新新闻: %s (ID: %s)", news.OriginalTitle, news.ID)
			newNews = append(newNews, news)
		}
	}
	
	// 如果没有新新闻，直接返回
	if len(newNews) == 0 {
		log.Printf("没有新的新闻需要处理")
		return nil
	}

	log.Printf("准备处理 %d 条新新闻", len(newNews))
	
	// 如果启用了AI分析
	if a.cfg.AI != nil && a.cfg.AI.Enabled {
		log.Printf("开始AI分析...")
		if err := a.analyzer.AnalyzeNews(ctx, newNews); err != nil {
			log.Printf("AI分析失败: %v", err)
			return err
		}
		log.Printf("AI分析完成")
	}
	
	// 处理每条新闻
	for _, news := range newNews {
		log.Printf("正在处理新闻: %s", news.OriginalTitle)
	
		// 发送新闻
		if err := a.sender.SendNews(ctx, news); err != nil {
			log.Printf("发送新闻失败: %v", err)
			continue
		}

		// 只有在成功发送后才更新缓存
		if err := a.cache.Set(news.Source, news.ID, true, time.Duration(a.cfg.Cache.TTL)*time.Second); err != nil {
			log.Printf("更新缓存失败: %v", err)
		} else {
			log.Printf("成功缓存新闻: %s (ID: %s), TTL: %d秒", news.OriginalTitle, news.ID, a.cfg.Cache.TTL)
		}

		log.Printf("成功处理新闻: %s", news.OriginalTitle)
	}

	return nil
}

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("启动新闻抓取服务...")

	// 创建数据目录
	dataDir := filepath.Join(os.Getenv("HOME"), ".news-fetcher")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}
	log.Printf("数据目录: %s", dataDir)

	// 加载配置
	cfg, err := config.LoadConfig("config/config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建应用实例
	app, err := NewApp(cfg)
	if err != nil {
		log.Fatalf("初始化应用失败: %v", err)
	}

	// 运行应用
	ctx := context.Background()
	if err := app.Run(ctx); err != nil {
		log.Fatalf("应用运行失败: %v", err)
	}
}
