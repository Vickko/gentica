package llm

import (
	"context"
)

// Session represents a chat session
type Session struct {
	ID                string
	Title             string
	SummaryMessageID  string
	Cost              float64
	CompletionTokens  int
	PromptTokens      int
}

// SessionService provides session management
type SessionService interface {
	Get(ctx context.Context, id string) (*Session, error)
	CreateTaskSession(ctx context.Context, taskID, parentID, title string) (*Session, error)
	Save(ctx context.Context, session *Session) error
}

// SimpleSessionService is a basic implementation of SessionService
type SimpleSessionService struct {
	sessions map[string]*Session
}

// NewSessionService creates a new session service
func NewSessionService() SessionService {
	return &SimpleSessionService{
		sessions: make(map[string]*Session),
	}
}

// Get retrieves a session by ID
func (s *SimpleSessionService) Get(ctx context.Context, id string) (*Session, error) {
	if session, ok := s.sessions[id]; ok {
		return session, nil
	}
	// Return a new session if not found
	return &Session{ID: id}, nil
}

// CreateTaskSession creates a new task session
func (s *SimpleSessionService) CreateTaskSession(ctx context.Context, taskID, parentID, title string) (*Session, error) {
	session := &Session{
		ID:    taskID,
		Title: title,
	}
	s.sessions[taskID] = session
	return session, nil
}

// Save saves a session
func (s *SimpleSessionService) Save(ctx context.Context, session *Session) error {
	s.sessions[session.ID] = session
	return nil
}