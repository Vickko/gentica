package agent

import (
	"context"
	"fmt"
	"sync"
)

// AgentChain 串行执行多个 Agent
type AgentChain struct {
	agents []Agent
	name   string
}

// NewAgentChain 创建新的 Agent 链
func NewAgentChain(name string, agents ...Agent) *AgentChain {
	return &AgentChain{
		name:   name,
		agents: agents,
	}
}

// Run 串行执行所有 Agent
func (ac *AgentChain) Run(ctx context.Context, input string) (string, error) {
	if len(ac.agents) == 0 {
		return "", fmt.Errorf("no agents in chain")
	}

	result := input
	for i, agent := range ac.agents {
		output, err := agent.Run(ctx, result)
		if err != nil {
			return "", fmt.Errorf("agent %d (%s) failed: %w", i, agent.Name(), err)
		}
		result = output
	}
	return result, nil
}

// Name 返回链的名称
func (ac *AgentChain) Name() string {
	return ac.name
}

// GetAgents 获取链中的所有 Agent
func (ac *AgentChain) GetAgents() []Agent {
	return ac.agents
}

// AgentPool 并行执行多个 Agent
type AgentPool struct {
	agents []Agent
	name   string
}

// NewAgentPool 创建新的 Agent 池
func NewAgentPool(name string, agents ...Agent) *AgentPool {
	return &AgentPool{
		name:   name,
		agents: agents,
	}
}

// Run 并行执行所有 Agent
func (ap *AgentPool) Run(ctx context.Context, input string) ([]string, error) {
	if len(ap.agents) == 0 {
		return nil, fmt.Errorf("no agents in pool")
	}

	results := make([]string, len(ap.agents))
	errors := make([]error, len(ap.agents))

	var wg sync.WaitGroup
	for i, agent := range ap.agents {
		wg.Add(1)
		go func(idx int, a Agent) {
			defer wg.Done()
			results[idx], errors[idx] = a.Run(ctx, input)
		}(i, agent)
	}
	wg.Wait()

	// 检查错误
	for i, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("agent %s failed: %w", ap.agents[i].Name(), err)
		}
	}

	return results, nil
}

// Name 返回池的名称
func (ap *AgentPool) Name() string {
	return ap.name
}

// GetAgents 获取池中的所有 Agent
func (ap *AgentPool) GetAgents() []Agent {
	return ap.agents
}

// AgentPoolResult 并行执行的详细结果
type AgentPoolResult struct {
	AgentName string
	Result    string
	Error     error
}

// RunWithDetails 并行执行并返回详细结果
func (ap *AgentPool) RunWithDetails(ctx context.Context, input string) []AgentPoolResult {
	results := make([]AgentPoolResult, len(ap.agents))

	var wg sync.WaitGroup
	for i, agent := range ap.agents {
		wg.Add(1)
		go func(idx int, a Agent) {
			defer wg.Done()
			result, err := a.Run(ctx, input)
			results[idx] = AgentPoolResult{
				AgentName: a.Name(),
				Result:    result,
				Error:     err,
			}
		}(i, agent)
	}
	wg.Wait()

	return results
}

// RouterFunc 路由函数，根据输入决定使用哪个 Agent
type RouterFunc func(ctx context.Context, input string) (string, error)

// AgentRouter 条件路由器
type AgentRouter struct {
	routes map[string]Agent
	router RouterFunc
	name   string
}

// NewAgentRouter 创建新的 Agent 路由器
func NewAgentRouter(name string, router RouterFunc) *AgentRouter {
	return &AgentRouter{
		name:   name,
		routes: make(map[string]Agent),
		router: router,
	}
}

// AddRoute 添加路由
func (ar *AgentRouter) AddRoute(key string, agent Agent) {
	ar.routes[key] = agent
}

// RemoveRoute 移除路由
func (ar *AgentRouter) RemoveRoute(key string) {
	delete(ar.routes, key)
}

// Run 根据路由执行相应的 Agent
func (ar *AgentRouter) Run(ctx context.Context, input string) (string, error) {
	// 使用路由函数决定使用哪个 Agent
	key, err := ar.router(ctx, input)
	if err != nil {
		return "", fmt.Errorf("router function failed: %w", err)
	}

	agent, ok := ar.routes[key]
	if !ok {
		return "", fmt.Errorf("no agent for route: %s", key)
	}

	return agent.Run(ctx, input)
}

// Name 返回路由器的名称
func (ar *AgentRouter) Name() string {
	return ar.name
}

// GetRoutes 获取所有路由
func (ar *AgentRouter) GetRoutes() map[string]Agent {
	return ar.routes
}

// AgentPipeline 更复杂的 Agent 管道，支持条件分支
type AgentPipeline struct {
	name  string
	steps []PipelineStep
}

// PipelineStep 管道步骤
type PipelineStep struct {
	Agent     Agent
	Condition func(ctx context.Context, input string, previousResult string) bool
	Transform func(previousResult string) string // 转换前一步的输出
}

// NewAgentPipeline 创建新的 Agent 管道
func NewAgentPipeline(name string) *AgentPipeline {
	return &AgentPipeline{
		name:  name,
		steps: make([]PipelineStep, 0),
	}
}

// AddStep 添加步骤
func (p *AgentPipeline) AddStep(agent Agent, condition func(context.Context, string, string) bool, transform func(string) string) *AgentPipeline {
	step := PipelineStep{
		Agent:     agent,
		Condition: condition,
		Transform: transform,
	}

	// 如果没有提供条件函数，默认总是执行
	if step.Condition == nil {
		step.Condition = func(context.Context, string, string) bool { return true }
	}

	// 如果没有提供转换函数，默认直接传递
	if step.Transform == nil {
		step.Transform = func(s string) string { return s }
	}

	p.steps = append(p.steps, step)
	return p
}

// Run 执行管道
func (p *AgentPipeline) Run(ctx context.Context, input string) (string, error) {
	if len(p.steps) == 0 {
		return "", fmt.Errorf("no steps in pipeline")
	}

	result := input
	for i, step := range p.steps {
		// 检查条件
		if !step.Condition(ctx, input, result) {
			continue // 跳过这个步骤
		}

		// 转换输入
		transformedInput := step.Transform(result)

		// 执行 Agent
		output, err := step.Agent.Run(ctx, transformedInput)
		if err != nil {
			return "", fmt.Errorf("pipeline step %d (%s) failed: %w", i, step.Agent.Name(), err)
		}

		result = output
	}

	return result, nil
}

// Name 返回管道的名称
func (p *AgentPipeline) Name() string {
	return p.name
}

// AgentOrchestrator 高级编排器，支持复杂的 Agent 协作模式
type AgentOrchestrator struct {
	name      string
	agents    map[string]Agent
	workflows map[string]func(context.Context, string) (string, error)
}

// NewAgentOrchestrator 创建新的编排器
func NewAgentOrchestrator(name string) *AgentOrchestrator {
	return &AgentOrchestrator{
		name:      name,
		agents:    make(map[string]Agent),
		workflows: make(map[string]func(context.Context, string) (string, error)),
	}
}

// RegisterAgent 注册 Agent
func (o *AgentOrchestrator) RegisterAgent(name string, agent Agent) {
	o.agents[name] = agent
}

// RegisterWorkflow 注册工作流
func (o *AgentOrchestrator) RegisterWorkflow(name string, workflow func(context.Context, string) (string, error)) {
	o.workflows[name] = workflow
}

// ExecuteWorkflow 执行工作流
func (o *AgentOrchestrator) ExecuteWorkflow(ctx context.Context, workflowName string, input string) (string, error) {
	workflow, ok := o.workflows[workflowName]
	if !ok {
		return "", fmt.Errorf("workflow %s not found", workflowName)
	}

	return workflow(ctx, input)
}

// GetAgent 获取已注册的 Agent
func (o *AgentOrchestrator) GetAgent(name string) (Agent, bool) {
	agent, ok := o.agents[name]
	return agent, ok
}