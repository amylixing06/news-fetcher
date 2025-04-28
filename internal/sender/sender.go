package sender

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/amylixing/news-fetcher/internal/cache"
	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Sender 消息发送器
type Sender struct {
	cfg    *config.TelegramConfig
	bot    *tgbotapi.BotAPI
	client *http.Client
	cache  *cache.Cache
}

// NewSender 创建新的发送器
func NewSender(cfg *config.TelegramConfig, cache *cache.Cache) (*Sender, error) {
	if cfg == nil || !cfg.Enabled {
		return &Sender{
			cache: cache,
		}, nil
	}

	// 配置HTTP客户端
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     30 * time.Second,
		},
		Timeout: time.Duration(cfg.Timeout) * time.Second,
	}

	// 如果配置了代理
	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("解析代理URL失败: %v", err)
		}
		httpClient.Transport.(*http.Transport).Proxy = http.ProxyURL(proxyURL)
	}

	// 创建Telegram机器人
	bot, err := tgbotapi.NewBotAPIWithClient(cfg.Bot.Token, tgbotapi.APIEndpoint, httpClient)
	if err != nil {
		return nil, fmt.Errorf("创建Telegram机器人失败: %v", err)
	}

	return &Sender{
		cfg:    cfg,
		bot:    bot,
		client: httpClient,
		cache:  cache,
	}, nil
}

// SendNews 发送新闻消息
func (s *Sender) SendNews(ctx context.Context, news *models.News) error {
	if s.bot == nil {
		log.Printf("[%s] Telegram 未启用，跳过发送: %s (ID: %v)", news.Source, news.OriginalTitle, news.ID)
		return nil
	}

	message := s.formatMessage(news)
	if message == "" {
		log.Printf("[%s] 消息内容为空，跳过发送: %s (ID: %v)", news.Source, news.OriginalTitle, news.ID)
		return fmt.Errorf("消息内容为空")
	}

	log.Printf("[%s] 准备发送新闻: %s (ID: %v)", news.Source, news.OriginalTitle, news.ID)

	var lastErr error
	for i := 0; i < s.cfg.Retry.Count; i++ {
		if i > 0 {
			log.Printf("[%s] 第%d次重试发送新闻: %s (ID: %v)", news.Source, i+1, news.OriginalTitle, news.ID)
			time.Sleep(time.Duration(s.cfg.Retry.Interval) * time.Second)
		}

		// 为每个聊天ID发送消息
		for _, chatID := range s.cfg.Bot.ChatIDs {
			log.Printf("[%s] 正在发送到聊天 %s: %s (ID: %v)", news.Source, chatID, news.OriginalTitle, news.ID)
			if err := s.sendToChat(ctx, chatID, message); err != nil {
				lastErr = err
				log.Printf("[%s] 发送到聊天 %s 失败: %s (ID: %v), 错误: %v", news.Source, chatID, news.OriginalTitle, news.ID, err)
				continue
			}
			log.Printf("[%s] 成功发送到聊天 %s: %s (ID: %v)", news.Source, chatID, news.OriginalTitle, news.ID)
		}

		if lastErr == nil {
			return nil
		}
	}

	return fmt.Errorf("发送新闻失败: %v", lastErr)
}

// sendToChat 发送消息到指定聊天
func (s *Sender) sendToChat(ctx context.Context, chatID string, message string) error {
	if chatID == "" {
		return fmt.Errorf("无效的聊天ID")
	}

	// 解析聊天ID
	parsedChatID := parseChatID(chatID)
	if parsedChatID == 0 {
		return fmt.Errorf("解析聊天ID失败: %s", chatID)
	}

	log.Printf("准备发送消息到聊天ID: %d", parsedChatID)
	log.Printf("消息内容: %s", message)

	msg := tgbotapi.NewMessage(parsedChatID, message)
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		log.Printf("正在调用Telegram API发送消息...")
		_, err := s.bot.Send(msg)
		if err != nil {
			log.Printf("发送消息到聊天 %d 失败: %v", parsedChatID, err)
			return fmt.Errorf("发送消息失败: %v", err)
		}
		log.Printf("成功发送消息到聊天 %d", parsedChatID)
		return nil
	}
}

// formatMessage 格式化消息
func (s *Sender) formatMessage(news *models.News) string {
	// 清理文本中的 HTML 标签
	cleanText := func(text string) string {
		text = strings.ReplaceAll(text, "<br>", "\n")
		text = strings.ReplaceAll(text, "<p>", "\n")
		text = strings.ReplaceAll(text, "</p>", "\n")
		text = strings.ReplaceAll(text, "<img", "[图片]")
		text = strings.ReplaceAll(text, "<a", "[链接]")
		text = strings.ReplaceAll(text, "</a>", "")
		return text
	}

	// 格式化消息
	message := fmt.Sprintf("📰 *%s*\n\n", news.OriginalTitle)
	message += fmt.Sprintf("�� 发布时间: %s\n", news.CreateTime.Format("2006-01-02 15:04"))
	message += fmt.Sprintf("🔗 原文链接: %s\n\n", news.Link)
	message += fmt.Sprintf("📝 内容摘要:\n%s\n\n", cleanText(news.OriginalContent))

	// 添加 AI 分析部分
	if news.Analysis != "" {
		message += "🤖 *AI 分析*\n\n"
		message += news.Analysis
	}

	return message
}

// parseChatID 解析聊天ID
func parseChatID(chatID string) int64 {
	log.Printf("开始解析聊天ID: %s", chatID)

	// 直接转换为int64
	id, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		log.Printf("解析聊天ID失败: %v, 原始值: %s", err, chatID)
		return 0
	}

	log.Printf("解析后的聊天ID (int64): %d", id)
	return id
}

// sendTestMessage 发送测试消息
func sendTestMessage(chatID int64) bool {
	// 创建机器人
	bot, err := tgbotapi.NewBotAPI("7597062287:AAFPJzK4aYK-_thgs9KfHW8yF16slkRfoVg")
	if err != nil {
		log.Printf("创建机器人失败: %v", err)
		return false
	}

	// 配置代理
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	if err != nil {
		log.Printf("解析代理URL失败: %v", err)
		return false
	}

	bot.Client = &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
		Timeout: time.Second * 30,
	}

	msg := tgbotapi.NewMessage(chatID, "测试消息")
	msg.ParseMode = tgbotapi.ModeHTML
	msg.DisableWebPagePreview = true

	_, err = bot.Send(msg)
	if err != nil {
		log.Printf("发送测试消息到聊天 %d 失败: %v", chatID, err)
		return false
	}

	log.Printf("发送测试消息到聊天 %d 成功", chatID)
	return true
}
