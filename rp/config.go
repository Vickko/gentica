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

	// 角色设定（用于 Character 和 Sidecar）
	CharacterSetting string `yaml:"character_setting" json:"character_setting"`

	// 角色指令（仅用于 Character Agent，例如回复格式、互动规则等）
	RoleInstruction string `yaml:"role_instruction" json:"role_instruction"`

	// 兼容旧字段：如果 yaml 里还是 system_prompt，我们会尝试自动拆分或保留
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

	// 兼容逻辑：如果 system_prompt 存在但 character_setting 为空
	// 这意味着是旧格式的配置文件。
	// 在这种情况下，我们暂时将 system_prompt 全部视为 character_setting，
	// 但这并不完美。理想情况下，应该更新 yaml 文件。
	if config.CharacterSetting == "" && config.SystemPrompt != "" {
		config.CharacterSetting = config.SystemPrompt
	}

	// 确保至少有 Character Setting
	if config.CharacterSetting == "" {
		return nil, fmt.Errorf("配置错误: character_setting (或 system_prompt) 字段不能为空")
	}

	return &config, nil
}
