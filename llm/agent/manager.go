package agent

import (
	"context"
	"fmt"
	"gentica/config"
	"gentica/csync"
	"gentica/llm/tools"
	"gentica/message"
	"gentica/session"
)

// AgentManager manages all agent instances
type AgentManager struct {
	agents *csync.Map[string, Service]
}

// NewAgentManager creates a new agent manager
func NewAgentManager() *AgentManager {
	return &AgentManager{
		agents: csync.NewMap[string, Service](),
	}
}

// Register registers an agent with the manager
func (m *AgentManager) Register(id string, agent Service) {
	m.agents.Set(id, agent)
}

// Get retrieves an agent by ID
func (m *AgentManager) Get(id string) (Service, bool) {
	return m.agents.Get(id)
}

// List returns all registered agent IDs
func (m *AgentManager) List() []string {
	ids := make([]string, 0)
	for k := range m.agents.Seq2() {
		ids = append(ids, k)
	}
	return ids
}

// CreateAgentTool creates a tool wrapper for an agent
func (m *AgentManager) CreateAgentTool(agentID string, sessions session.Service, messages message.Service) (tools.BaseTool, error) {
	agent, ok := m.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent %s not found", agentID)
	}
	return NewAgentToolWithID(agentID, agent, sessions, messages), nil
}

// InitializeAgents initializes all agents from config
func InitializeAgents(
	ctx context.Context,
	cfg config.Config,
	sessions session.Service,
	messages message.Service,
) (*AgentManager, error) {
	manager := NewAgentManager()
	
	// First pass: create all agents without agent tools
	agents := make(map[string]Service)
	for id, agentCfg := range cfg.Agents {
		if agentCfg.Disabled {
			continue
		}
		
		agent, err := NewAgent(ctx, agentCfg, sessions, messages, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create agent %s: %w", id, err)
		}
		agents[id] = agent
		manager.Register(id, agent)
	}
	
	// Second pass: create agent tools for agents that need them
	for id, agentCfg := range cfg.Agents {
		if agentCfg.Disabled {
			continue
		}
		
		// Check if this agent is allowed to call other agents
		if agentCfg.AllowedAgents != nil && len(agentCfg.AllowedAgents) > 0 {
			agentTools := make(map[string]tools.BaseTool)
			
			for _, allowedAgentID := range agentCfg.AllowedAgents {
				if targetAgent, ok := agents[allowedAgentID]; ok {
					// Create a tool for each allowed agent
					tool := NewAgentToolWithID(allowedAgentID, targetAgent, sessions, messages)
					agentTools[fmt.Sprintf("agent_%s", allowedAgentID)] = tool
				}
			}
			
			// Recreate the agent with agent tools
			if len(agentTools) > 0 {
				newAgent, err := NewAgent(ctx, agentCfg, sessions, messages, agentTools)
				if err != nil {
					return nil, fmt.Errorf("failed to recreate agent %s with tools: %w", id, err)
				}
				agents[id] = newAgent
				manager.Register(id, newAgent)
			}
		}
	}
	
	return manager, nil
}

// NewAgentToolWithID creates an agent tool with a specific ID
func NewAgentToolWithID(
	agentID string,
	agent Service,
	sessions session.Service,
	messages message.Service,
) tools.BaseTool {
	return &agentToolWithID{
		agentTool: agentTool{
			agent:    agent,
			sessions: sessions,
			messages: messages,
		},
		agentID: agentID,
	}
}

// agentToolWithID extends agentTool with a specific ID
type agentToolWithID struct {
	agentTool
	agentID string
}

// Name returns the unique name for this agent tool
func (t *agentToolWithID) Name() string {
	return fmt.Sprintf("agent_%s", t.agentID)
}

// Info returns the tool information with a unique name
func (t *agentToolWithID) Info() tools.ToolInfo {
	info := t.agentTool.Info()
	info.Name = t.Name()
	info.Description = fmt.Sprintf("Launch the %s agent. %s", t.agentID, info.Description)
	return info
}