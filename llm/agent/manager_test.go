package agent

import (
	"context"
	"gentica/message"
	"gentica/pubsub"
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestAgentManager(t *testing.T) {
	// Test AgentManager creation
	manager := NewAgentManager()
	if manager == nil {
		t.Fatal("Failed to create AgentManager")
	}

	// Test agent registration
	mockAgent := &mockService{}
	manager.Register("test", mockAgent)

	// Test agent retrieval
	_, ok := manager.Get("test")
	if !ok {
		t.Error("Failed to retrieve registered agent")
	}

	// Test listing agents
	ids := manager.List()
	found := false
	for _, id := range ids {
		if id == "test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("List() didn't return expected agent ID")
	}

	// Test getting non-existent agent
	_, ok = manager.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for non-existent agent")
	}
}

// mockService is a minimal implementation of Service interface for testing
type mockService struct{}

func (m *mockService) Subscribe(ctx context.Context) <-chan pubsub.Event[AgentEvent] {
	ch := make(chan pubsub.Event[AgentEvent])
	close(ch)
	return ch
}

func (m *mockService) Model() catwalk.Model {
	return catwalk.Model{}
}

func (m *mockService) Run(ctx context.Context, sessionID string, content string, attachments ...message.Attachment) (<-chan AgentEvent, error) {
	ch := make(chan AgentEvent)
	close(ch)
	return ch, nil
}

func (m *mockService) Cancel(sessionID string) {}

func (m *mockService) CancelAll() {}

func (m *mockService) IsSessionBusy(sessionID string) bool {
	return false
}

func (m *mockService) IsBusy() bool {
	return false
}

func (m *mockService) Summarize(ctx context.Context, sessionID string) error {
	return nil
}

func (m *mockService) UpdateModel() error {
	return nil
}

func (m *mockService) QueuedPrompts(sessionID string) int {
	return 0
}

func (m *mockService) ClearQueue(sessionID string) {}

func TestAgentToolWithID(t *testing.T) {
	// Test that agent tools have unique names
	mockAgent := &mockService{}
	tool1 := NewAgentToolWithID("agent1", mockAgent, nil, nil)
	tool2 := NewAgentToolWithID("agent2", mockAgent, nil, nil)

	if tool1.Name() == tool2.Name() {
		t.Error("Agent tools should have unique names")
	}

	if tool1.Name() != "agent_agent1" {
		t.Errorf("Expected tool name 'agent_agent1', got '%s'", tool1.Name())
	}

	if tool2.Name() != "agent_agent2" {
		t.Errorf("Expected tool name 'agent_agent2', got '%s'", tool2.Name())
	}
}
