package translator

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"cloud.google.com/go/translate"
	"github.com/amylixing/news-fetcher/internal/config"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

// GoogleTranslator Google翻译实现
type GoogleTranslator struct {
	client         *translate.Client
	targetLanguage language.Tag
	timeout        time.Duration
	cfg            config.TranslatorConfig
}

// NewGoogleTranslator 创建Google翻译实例
func NewGoogleTranslator(cfg config.TranslatorConfig) *GoogleTranslator {
	return &GoogleTranslator{
		targetLanguage: language.Make(cfg.TargetLanguage),
		timeout:        time.Duration(cfg.Timeout) * time.Second,
		cfg:            cfg,
	}
}

// Init 初始化Google翻译
func (t *GoogleTranslator) Init() error {
	ctx, cancel := context.WithTimeout(context.Background(), t.timeout)
	defer cancel()

	// 配置HTTP客户端
	proxyURL, err := url.Parse(t.cfg.ProxyURL)
	if err != nil {
		return fmt.Errorf("解析代理URL失败: %v", err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   t.timeout,
	}

	client, err := translate.NewClient(ctx, option.WithAPIKey(t.cfg.APIKey), option.WithHTTPClient(httpClient))
	if err != nil {
		return fmt.Errorf("创建Google翻译客户端失败: %v", err)
	}

	t.client = client
	return nil
}

// Translate 翻译文本
func (t *GoogleTranslator) Translate(ctx context.Context, text string) (string, error) {
	if t.client == nil {
		return "", fmt.Errorf("Google翻译客户端未初始化")
	}

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// 翻译文本
	translations, err := t.client.Translate(ctx, []string{text}, t.targetLanguage, nil)
	if err != nil {
		return "", fmt.Errorf("翻译失败: %v", err)
	}

	if len(translations) == 0 {
		return "", fmt.Errorf("未获取到翻译结果")
	}

	return translations[0].Text, nil
}

// GetName 获取翻译服务名称
func (t *GoogleTranslator) GetName() string {
	return "google"
}

// Close 关闭翻译客户端
func (t *GoogleTranslator) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}
