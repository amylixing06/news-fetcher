package main

import (
	"context"
	"log"
	"time"

	"github.com/amylixing/news-fetcher/internal/config"
	"github.com/amylixing/news-fetcher/internal/models"
	"github.com/amylixing/news-fetcher/internal/sender"
)

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 创建发送器
	s, err := sender.NewSender(cfg.Telegram, nil)
	if err != nil {
		log.Fatalf("创建发送器失败: %v", err)
	}

	// 测试消息
	testMessage := "测试消息\n" +
		"时间: " + time.Now().Format("2006-01-02 15:04:05") + "\n" +
		"这是一条测试消息，用于验证消息发送功能是否正常工作。"

	// 发送到所有配置的聊天
	for _, chatID := range cfg.Telegram.Bot.ChatIDs {
		log.Printf("正在发送测试消息到聊天: %s", chatID)
		err := s.SendNews(context.Background(), &models.News{
			OriginalTitle:   "测试消息",
			OriginalContent: testMessage,
			Source:          "测试",
			CreateTime:      time.Now(),
		})
		if err != nil {
			log.Printf("发送到聊天 %s 失败: %v", chatID, err)
		} else {
			log.Printf("成功发送到聊天 %s", chatID)
		}
	}
}
