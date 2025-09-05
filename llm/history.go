package llm

// HistoryService provides history management
type HistoryService interface {
	// Add any needed methods here
}

// SimpleHistoryService is a basic implementation of HistoryService
type SimpleHistoryService struct{}

// NewHistoryService creates a new history service
func NewHistoryService() HistoryService {
	return &SimpleHistoryService{}
}