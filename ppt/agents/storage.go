package agents

import (
	"sync"
)

// StoredSearchResult 存储的搜索结果，包含原始内容和评分
type StoredSearchResult struct {
	SearchResult
	Score      float64 `json:"score"`
	Reasoning  string  `json:"reasoning"`
	SearchRound int     `json:"search_round"`
}

// QualityResult 高质量结果
type QualityResult struct {
	Title      string  `json:"title"`
	Link       string  `json:"link"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	Reasoning  string  `json:"reasoning"`
}

// SearchStorage 搜索结果存储管理器
type SearchStorage struct {
	mu sync.RWMutex
	// 所有搜索结果，以title为key
	searchResults map[string]*StoredSearchResult
	// 高质量结果集
	qualityResults []QualityResult
	// 统计信息
	totalSearches int
	searchRounds  int
}

// NewSearchStorage 创建新的存储管理器
func NewSearchStorage() *SearchStorage {
	return &SearchStorage{
		searchResults:  make(map[string]*StoredSearchResult),
		qualityResults: make([]QualityResult, 0),
		totalSearches:  0,
		searchRounds:   0,
	}
}

// StoreSearchResult 存储搜索结果
func (s *SearchStorage) StoreSearchResult(result SearchResult, score float64, reasoning string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 如果已存在相同标题的结果，保留分数较高的
	if existing, ok := s.searchResults[result.Title]; ok {
		if score > existing.Score {
			s.searchResults[result.Title] = &StoredSearchResult{
				SearchResult: result,
				Score:        score,
				Reasoning:    reasoning,
				SearchRound:   s.searchRounds,
			}
		}
	} else {
		s.searchResults[result.Title] = &StoredSearchResult{
			SearchResult: result,
			Score:        score,
			Reasoning:    reasoning,
			SearchRound:   s.searchRounds,
		}
	}
	s.totalSearches++
}

// AddToQualitySet 添加到高质量结果集
func (s *SearchStorage) AddToQualitySet(title string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	result, exists := s.searchResults[title]
	if !exists {
		return false
	}
	
	// 检查是否已在质量集中
	for _, qr := range s.qualityResults {
		if qr.Title == title {
			return false // 已存在
		}
	}
	
	// 添加到质量集
	s.qualityResults = append(s.qualityResults, QualityResult{
		Title:     result.Title,
		Link:      result.Link,
		Content:   result.Content,
		Score:     result.Score,
		Reasoning: result.Reasoning,
	})
	
	return true
}

// GetQualityResults 获取高质量结果集
func (s *SearchStorage) GetQualityResults() []QualityResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回副本以防止外部修改
	results := make([]QualityResult, len(s.qualityResults))
	copy(results, s.qualityResults)
	return results
}

// GetAllSearchResults 获取所有搜索结果
func (s *SearchStorage) GetAllSearchResults() map[string]*StoredSearchResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// 返回副本
	results := make(map[string]*StoredSearchResult)
	for k, v := range s.searchResults {
		results[k] = v
	}
	return results
}

// GetSearchResultByTitle 根据标题获取搜索结果
func (s *SearchStorage) GetSearchResultByTitle(title string) (*StoredSearchResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result, exists := s.searchResults[title]
	return result, exists
}

// GetStats 获取统计信息
func (s *SearchStorage) GetStats() (totalSearches, searchRounds, qualityCount int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.totalSearches, s.searchRounds, len(s.qualityResults)
}

// IncrementRound 增加搜索轮数
func (s *SearchStorage) IncrementRound() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.searchRounds++
}

// ClearQualityResults 清空高质量结果集
func (s *SearchStorage) ClearQualityResults() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.qualityResults = []QualityResult{}
}

// Reset 重置所有数据
func (s *SearchStorage) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.searchResults = make(map[string]*StoredSearchResult)
	s.qualityResults = []QualityResult{}
	s.totalSearches = 0
	s.searchRounds = 0
}

// GetQualityCount 获取高质量结果数量
func (s *SearchStorage) GetQualityCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.qualityResults)
}