package agents

// QueryAnalysisResult 定义查询分析结果的结构
type QueryAnalysisResult struct {
	NeedSearch bool   `json:"need_search"`
	Reason     string `json:"reason"`
	QueryType  string `json:"query_type"`
}

// SearchEvaluationRequest 定义搜索评估请求的结构
type SearchEvaluationRequest struct {
	ResearchTopic string         `json:"research_topic"`
	SearchKeyword string         `json:"search_keyword"`
	SearchResults []SearchResult `json:"search_results"`
}

// SearchResult 定义单个搜索结果的结构
type SearchResult struct {
	Title       string `json:"title"`
	Link        string `json:"link"`
	Content     string `json:"content"`      // 完整的文章内容
	PublishDate string `json:"publish_date,omitempty"`
	Source      string `json:"source,omitempty"`      // 来源网站
}

// SearchEvaluationResult 定义搜索评估结果的结构
type SearchEvaluationResult struct {
	ResearchTopic              string                    `json:"research_topic"`
	SearchKeyword              string                    `json:"search_keyword"`
	TopicTimelinessSensitivity string                    `json:"topic_timeliness_sensitivity"`
	Results                    []EvaluatedSearchResult   `json:"results"`
	Summary                    string                    `json:"summary"`
}

// EvaluatedSearchResult 定义评估后的单个搜索结果
type EvaluatedSearchResult struct {
	Title           string  `json:"title"`
	RelevanceScore  float64 `json:"relevance_score"`
	QualityScore    float64 `json:"quality_score"`
	TimelinessScore float64 `json:"timeliness_score"`
	TotalScore      float64 `json:"total_score"`
	Reasoning       string  `json:"reasoning"`
}