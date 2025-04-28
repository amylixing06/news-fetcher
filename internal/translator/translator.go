package translator

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"cloud.google.com/go/translate"
	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/models"
	"golang.org/x/text/language"
	"google.golang.org/api/option"
)

var translatorConfig config.TranslatorConfig

// InitTranslator 初始化翻译器配置
func InitTranslator(cfg config.TranslatorConfig) {
	translatorConfig = cfg
}

// TranslationProvider 翻译服务提供商接口
type TranslationProvider interface {
	Translate(ctx context.Context, text string) (string, error)
	GetName() string
}

// Translator 翻译器
type Translator struct {
	client         *translate.Client
	targetLanguage language.Tag
	timeout        time.Duration
	cfg            config.TranslatorConfig
}

// NewTranslator 创建新的翻译器
func NewTranslator(cfg config.TranslatorConfig) *Translator {
	return &Translator{
		targetLanguage: language.Make(cfg.TargetLanguage),
		timeout:        time.Duration(cfg.Timeout) * time.Second,
		cfg:            cfg,
	}
}

// Init 初始化翻译器
func (t *Translator) Init() error {
	// 配置HTTP客户端
	client := &http.Client{
		Timeout: t.timeout,
	}

	if t.cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(t.cfg.ProxyURL)
		if err != nil {
			return fmt.Errorf("解析代理URL失败: %v", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	// 创建翻译客户端
	ctx := context.Background()
	translateClient, err := translate.NewClient(ctx, option.WithAPIKey(t.cfg.APIKey), option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("创建翻译客户端失败: %v", err)
	}

	t.client = translateClient
	return nil
}

// Close 关闭翻译器
func (t *Translator) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}

// TranslateNews 翻译新闻
func (t *Translator) TranslateNews(ctx context.Context, newsList []models.News) ([]models.News, error) {
	if !t.cfg.Enabled || len(newsList) == 0 {
		fmt.Printf("翻译功能未启用或没有新闻需要翻译\n")
		return newsList, nil
	}

	fmt.Printf("开始翻译 %d 条新闻\n", len(newsList))

	// 创建超时上下文
	ctx, cancel := context.WithTimeout(ctx, t.timeout)
	defer cancel()

	// 准备要翻译的文本
	texts := make([]string, 0, len(newsList)*2)
	for _, news := range newsList {
		texts = append(texts, news.OriginalTitle, news.OriginalContent)
	}

	fmt.Printf("准备翻译文本，共 %d 段\n", len(texts))

	// 批量翻译
	translations, err := t.client.Translate(ctx, texts, t.targetLanguage, nil)
	if err != nil {
		fmt.Printf("翻译失败: %v\n", err)
		return newsList, fmt.Errorf("翻译失败: %v", err)
	}

	fmt.Printf("翻译完成，共 %d 段文本\n", len(translations))

	// 更新新闻内容
	translatedNews := make([]models.News, len(newsList))
	for i := 0; i < len(newsList); i++ {
		translatedNews[i] = newsList[i]
		translatedNews[i].TranslatedTitle = translations[i*2].Text
		translatedNews[i].TranslatedContent = translations[i*2+1].Text
	}

	fmt.Printf("新闻翻译完成\n")
	return translatedNews, nil
}

// translateToChinese 使用百度翻译API将英文翻译为中文
func translateToChinese(text string) (string, error) {
	if translatorConfig.Type != "baidu" {
		return text, fmt.Errorf("不支持的翻译服务类型: %s", translatorConfig.Type)
	}

	// 准备请求参数
	salt := strconv.FormatInt(time.Now().Unix(), 10)
	sign := generateBaiduSign(translatorConfig.AppID, text, salt, translatorConfig.SecretKey)

	// 构建请求URL
	apiURL := "https://api.fanyi.baidu.com/api/trans/vip/translate"
	params := url.Values{}
	params.Set("q", text)
	params.Set("from", "en")
	params.Set("to", "zh")
	params.Set("appid", translatorConfig.AppID)
	params.Set("salt", salt)
	params.Set("sign", sign)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: time.Duration(translatorConfig.Timeout) * time.Second,
	}

	// 发送请求
	resp, err := client.Get(apiURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("请求翻译API失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var result struct {
		From        string `json:"from"`
		To          string `json:"to"`
		TransResult []struct {
			Src string `json:"src"`
			Dst string `json:"dst"`
		} `json:"trans_result"`
		ErrorCode string `json:"error_code"`
		ErrorMsg  string `json:"error_msg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查错误
	if result.ErrorCode != "" {
		return "", fmt.Errorf("翻译API错误: %s - %s", result.ErrorCode, result.ErrorMsg)
	}

	// 返回翻译结果
	if len(result.TransResult) > 0 {
		return result.TransResult[0].Dst, nil
	}

	return "", fmt.Errorf("未收到翻译结果")
}

// generateBaiduSign 生成百度翻译API签名
func generateBaiduSign(appID, query, salt, secretKey string) string {
	str := appID + query + salt + secretKey
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
