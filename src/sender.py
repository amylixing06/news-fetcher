def formatMessage(article, ai_analysis=None):
    # 提取文章信息
    title = article.get('title', '')
    description = article.get('description', '')
    url = article.get('url', '')
    source = article.get('source', {}).get('name', '')
    publishedAt = article.get('publishedAt', '')
    
    # 格式化日期
    try:
        date = datetime.fromisoformat(publishedAt.replace('Z', '+00:00'))
        formatted_date = date.strftime('%Y-%m-%d %H:%M')
    except:
        formatted_date = publishedAt
    
    # 构建消息
    message = f"<b>{title}</b>\n\n"
    message += f"{description}\n\n"
    message += f"来源: {source}\n"
    message += f"发布时间: {formatted_date}\n"
    message += f"<a href='{url}'>阅读原文</a>\n\n"
    
    # 添加 AI 分析
    if ai_analysis:
        message += f"<b>AI 分析</b>\n"
        message += f"{ai_analysis}\n\n"
    
    return message 