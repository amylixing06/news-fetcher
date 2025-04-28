package ai

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/models"
	"github.com/sashabaranov/go-openai"
)

type Analyzer struct {
	cfg    *config.AIConfig
	client *openai.Client
}

func NewAnalyzer(cfg *config.AIConfig) (*Analyzer, error) {
	if !cfg.Enabled {
		return &Analyzer{cfg: cfg}, nil
	}

	log.Printf("正在初始化AI分析器，使用模型: %s", cfg.Model)

	// 配置HTTP代理
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	if err != nil {
		return nil, fmt.Errorf("解析代理URL失败: %v", err)
	}

	// 创建HTTP客户端
	httpClient := &http.Client{
		Transport: &http.Transport{
			Proxy:               http.ProxyURL(proxyURL),
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
		},
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}

	// 初始化OpenAI客户端
	config := openai.DefaultConfig(cfg.APIKey)
	config.BaseURL = "https://api.deepseek.com/v1"
	config.HTTPClient = httpClient
	client := openai.NewClientWithConfig(config)

	return &Analyzer{
		cfg:    cfg,
		client: client,
	}, nil
}

func (a *Analyzer) AnalyzeNews(ctx context.Context, newsList []*models.News) error {
	if !a.cfg.Enabled || len(newsList) == 0 {
		return nil
	}

	for _, news := range newsList {
		if err := a.analyzeNewsItem(ctx, news); err != nil {
			log.Printf("分析新闻失败 [%s]: %v", news.ID, err)
			continue
		}
	}

	return nil
}

func (a *Analyzer) analyzeNewsItem(ctx context.Context, news *models.News) error {
	log.Printf("开始分析新闻: %s (ID: %s)", news.OriginalTitle, news.ID)

	// 创建带超时的上下文
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(a.cfg.Timeout)*time.Second)
	defer cancel()

	// 构建提示词
	prompt := a.buildPrompt(news)

	// 设置重试
	var lastErr error
	for i := 0; i < a.cfg.Retry.Count; i++ {
		if i > 0 {
			log.Printf("第%d次重试分析新闻: %s", i+1, news.OriginalTitle)
			time.Sleep(time.Duration(a.cfg.Retry.Interval) * time.Second)
		}

		resp, err := a.client.CreateChatCompletion(
			timeoutCtx,
			openai.ChatCompletionRequest{
				Model: a.cfg.Model,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: prompt,
					},
				},
				MaxTokens:   a.cfg.Params.MaxTokens,
				Temperature: float32(a.cfg.Params.Temperature),
			},
		)

		if err != nil {
			lastErr = fmt.Errorf("AI请求失败: %v", err)
			continue
		}

		if len(resp.Choices) > 0 {
			news.TranslatedContent = resp.Choices[0].Message.Content
			log.Printf("新闻分析完成: %s", news.OriginalTitle)
			return nil
		}
	}

	return lastErr
}

func (a *Analyzer) buildPrompt(news *models.News) string {
	return fmt.Sprintf(`请分析以下新闻：

标题：%s
内容：%s

请提供以下分析：
1. 新闻类型（政治、经济、科技、加密货币等）
2. 影响范围（局部、区域、全球）
3. 重要性评估（低、中、高）
4. 潜在影响（市场、政策、技术等）
5. 建议行动（关注、观望、采取行动等）

请用简洁明了的语言进行分析，每个部分用1-2句话说明。`, news.OriginalTitle, news.OriginalContent)
}
