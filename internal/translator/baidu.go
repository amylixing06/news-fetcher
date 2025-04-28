package translator

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/amylixing/news-fetcher/internal/config"
)

// BaiduTranslator 百度翻译实现
type BaiduTranslator struct {
	appID     string
	secretKey string
	timeout   time.Duration
	cfg       config.TranslatorConfig
}

// NewBaiduTranslator 创建百度翻译实例
func NewBaiduTranslator(cfg config.TranslatorConfig) *BaiduTranslator {
	return &BaiduTranslator{
		appID:     cfg.AppID,
		secretKey: cfg.SecretKey,
		timeout:   time.Duration(cfg.Timeout) * time.Second,
		cfg:       cfg,
	}
}

// Init 初始化百度翻译
func (t *BaiduTranslator) Init() error {
	if t.appID == "" || t.secretKey == "" {
		return fmt.Errorf("百度翻译AppID或SecretKey未配置")
	}
	return nil
}

// Translate 翻译文本
func (t *BaiduTranslator) Translate(ctx context.Context, text string) (string, error) {
	// 准备请求参数
	salt := strconv.FormatInt(time.Now().Unix(), 10)
	sign := generateSign(t.appID, text, salt, t.secretKey)

	// 构建请求URL
	apiURL := "https://api.fanyi.baidu.com/api/trans/vip/translate"
	params := url.Values{}
	params.Set("q", text)
	params.Set("from", "auto")
	params.Set("to", t.cfg.TargetLanguage)
	params.Set("appid", t.appID)
	params.Set("salt", salt)
	params.Set("sign", sign)

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: t.timeout,
	}

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL+"?"+params.Encode(), nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := ioutil.ReadAll(resp.Body)
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

// GetName 获取翻译服务名称
func (t *BaiduTranslator) GetName() string {
	return "baidu"
}

// Close 关闭翻译客户端
func (t *BaiduTranslator) Close() error {
	return nil
}

// generateSign 生成百度翻译API签名
func generateSign(appID, query, salt, secretKey string) string {
	str := appID + query + salt + secretKey
	sum := md5.Sum([]byte(str))
	return hex.EncodeToString(sum[:])
}
