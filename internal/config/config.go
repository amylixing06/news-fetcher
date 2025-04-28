package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config 应用配置
type Config struct {
	Sources  *SourcesConfig  `yaml:"sources"`
	Telegram *TelegramConfig `yaml:"telegram"`
	AI       *AIConfig       `yaml:"ai"`
	Cache    *CacheConfig    `yaml:"cache"`
	App      *AppConfig      `yaml:"app"`
}

// SourcesConfig 数据源配置
type SourcesConfig struct {
	API []*SourceConfig `yaml:"api"`
	RSS []*SourceConfig `yaml:"rss"`
}

// SourceConfig 单个数据源配置
type SourceConfig struct {
	URL      string                 `yaml:"url"`
	Params   map[string]interface{} `yaml:"params"`
	Headers  map[string]string      `yaml:"headers"`
	Retry    *RetryConfig           `yaml:"retry"`
	Timeout  int                    `yaml:"timeout"`
	ProxyURL string                 `yaml:"proxy_url"`
}

// TelegramConfig Telegram配置
type TelegramConfig struct {
	Enabled  bool         `yaml:"enabled"`
	Bot      *BotConfig   `yaml:"bot"`
	Retry    *RetryConfig `yaml:"retry"`
	Timeout  int          `yaml:"timeout"`
	ProxyURL string       `yaml:"proxy_url"`
}

// BotConfig 机器人配置
type BotConfig struct {
	Token   string   `yaml:"token"`
	ChatIDs []string `yaml:"chat_ids"`
}

// AIConfig AI配置
type AIConfig struct {
	Enabled  bool         `yaml:"enabled"`
	Provider string       `yaml:"provider"`
	Model    string       `yaml:"model"`
	APIKey   string       `yaml:"api_key"`
	Params   *AIParams    `yaml:"params"`
	Timeout  int          `yaml:"timeout"`
	Retry    *RetryConfig `yaml:"retry"`
}

// AIParams AI参数
type AIParams struct {
	MaxTokens   int     `yaml:"max_tokens"`
	Temperature float64 `yaml:"temperature"`
}

// CacheConfig 缓存配置
type CacheConfig struct {
	Type string `yaml:"type"`
	TTL  int    `yaml:"ttl"`
}

// AppConfig 应用配置
type AppConfig struct {
	FetchInterval int    `yaml:"fetch_interval"`
	LogLevel      string `yaml:"log_level"`
}

// RetryConfig 重试配置
type RetryConfig struct {
	Count    int `yaml:"count"`
	Interval int `yaml:"interval"`
}

// TranslatorConfig 翻译配置
type TranslatorConfig struct {
	Enabled        bool   `yaml:"enabled"`
	Type           string `yaml:"type"`
	APIKey         string `yaml:"api_key"`
	AppID          string `yaml:"app_id"`
	SecretKey      string `yaml:"secret_key"`
	TargetLanguage string `yaml:"target_language"`
	Timeout        int    `yaml:"timeout"`
	ProxyURL       string `yaml:"proxy_url"`
}

// LoadConfig 从文件加载配置
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("配置验证失败: %v", err)
	}

	return &cfg, nil
}

// validateConfig 验证配置
func validateConfig(cfg *Config) error {
	if cfg.Sources == nil || (len(cfg.Sources.API) == 0 && len(cfg.Sources.RSS) == 0) {
		return fmt.Errorf("未配置数据源")
	}

	if cfg.Telegram == nil || cfg.Telegram.Bot == nil {
		return fmt.Errorf("未配置Telegram")
	}

	if cfg.Telegram.Bot.Token == "" {
		return fmt.Errorf("未配置Telegram机器人令牌")
	}

	if len(cfg.Telegram.Bot.ChatIDs) == 0 {
		return fmt.Errorf("未配置Telegram聊天ID")
	}

	if cfg.AI != nil && cfg.AI.Enabled {
		if cfg.AI.APIKey == "" {
			return fmt.Errorf("未配置AI API密钥")
		}
		if cfg.AI.Model == "" {
			return fmt.Errorf("未配置AI模型")
		}
	}

	if cfg.App == nil {
		return fmt.Errorf("未配置应用参数")
	}

	if cfg.App.FetchInterval <= 0 {
		return fmt.Errorf("抓取间隔必须大于0")
	}

	return nil
}
