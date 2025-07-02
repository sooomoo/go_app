package pkg

type MQTopic string

const (
	MQTopicSearchKeywords         MQTopic = "search_keywords"          // 关键词搜索
	MQTopicSearchKeywordsProgress MQTopic = "search_keywords_progress" // 关键词搜索进度
	MQTopicSearchKeywordsResult   MQTopic = "search_keywords_result"   // 关键词搜索结果
)
