package pkg

type MQTopic string

const (
	MQTopicSearchKeywords         MQTopic = "search_keywords"          // 关键词搜索
	MQTopicSearchKeywordsProgress MQTopic = "search_keywords_progress" // 关键词搜索进度, progress 为 1 时为结果关键词搜索结果
)
