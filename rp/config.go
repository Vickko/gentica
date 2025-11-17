package rp

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RoleConfig 定义角色配置结构
type RoleConfig struct {
	// 角色名称
	Name string `yaml:"name" json:"name"`

	// 系统提示词（角色设定）
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`
}

// LoadRoleConfig 从指定路径加载角色配置
func LoadRoleConfig(configPath string) (*RoleConfig, error) {
	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析 YAML
	var config RoleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证必填字段
	if config.Name == "" {
		return nil, fmt.Errorf("配置错误: name 字段不能为空")
	}
	if config.SystemPrompt == "" {
		return nil, fmt.Errorf("配置错误: system_prompt 字段不能为空")
	}

	return &config, nil
}
