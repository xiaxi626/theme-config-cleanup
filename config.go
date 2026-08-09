package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// systemKeys 是 Gridea Pro 引擎在渲染时注入到 customConfig 的 key，
// 不属于任何主题 schema 定义，必须保留。
var systemKeys = map[string]bool{
	"avatar": true,
	"links":  true,
}

// ThemeSchema 对应主题文件夹下 config.json 的结构
type ThemeSchema struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Engine       string            `json:"engine,omitempty"`
	Author       string            `json:"author,omitempty"`
	CustomConfig []ThemeConfigItem `json:"customConfig"`
}

// ThemeConfigItem 主题配置项定义
type ThemeConfigItem struct {
	Name  string      `json:"name"`
	Label string      `json:"label"`
	Group string      `json:"group"`
	Value interface{} `json:"value"`
	Type  string      `json:"type"`
	Note  string      `json:"note,omitempty"`
	Card  string      `json:"card,omitempty"`
}

// CleanupPlan 清理计划文件结构
type CleanupPlan struct {
	ThemeName     string                 `json:"themeName"`
	ScanTime      string                 `json:"scanTime"`
	ExclusiveKeys []string               `json:"exclusiveKeys"`
	SharedKeys    []string               `json:"sharedKeys"`
	CurrentValues map[string]interface{} `json:"currentValues"`
}

// LoadConfig 读取 config.json 为 map，保留所有字段
func LoadConfig(configPath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}
	return config, nil
}

// SaveConfig 将 config map 写回文件，2 空格缩进
func SaveConfig(configPath string, config map[string]interface{}) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}
	return nil
}

// BackupConfig 备份 config.json 到 config.json.bak
func BackupConfig(configPath string) error {
	src, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(configPath + ".bak")
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

// LoadThemeSchema 读取主题的 config.json schema
func LoadThemeSchema(themesDir, themeName string) (*ThemeSchema, error) {
	schemaPath := filepath.Join(themesDir, themeName, "config.json")
	data, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}
	var schema ThemeSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// ExtractThemeKeys 从主题 schema 中提取配置项名称集合
func ExtractThemeKeys(schema *ThemeSchema) map[string]bool {
	keys := make(map[string]bool)
	for _, item := range schema.CustomConfig {
		keys[item.Name] = true
	}
	return keys
}

// CollectOtherThemeKeys 读取除目标主题外所有主题的 key 名
// 返回: key 集合, 主题名列表, error
func CollectOtherThemeKeys(themesDir, excludeName string) (map[string]bool, []string, error) {
	keys := make(map[string]bool)
	var themeNames []string

	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return nil, nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == excludeName {
			continue
		}
		schema, err := LoadThemeSchema(themesDir, entry.Name())
		if err != nil {
			continue // 跳过无法解析的主题
		}
		for k := range ExtractThemeKeys(schema) {
			keys[k] = true
		}
		themeNames = append(themeNames, entry.Name())
	}

	return keys, themeNames, nil
}

// GetCustomConfig 从 config map 中提取 customConfig
func GetCustomConfig(config map[string]interface{}) map[string]interface{} {
	if cc, ok := config["customConfig"].(map[string]interface{}); ok {
		return cc
	}
	return make(map[string]interface{})
}

// SavePlan 保存清理计划到工作区
func SavePlan(workDir string, plan *CleanupPlan) error {
	planPath := filepath.Join(workDir, ".theme-cleanup-plan.json")
	plan.ScanTime = time.Now().Format(time.RFC3339)
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(planPath, data, 0644)
}

// LoadPlan 从工作区加载清理计划
func LoadPlan(workDir string) (*CleanupPlan, error) {
	planPath := filepath.Join(workDir, ".theme-cleanup-plan.json")
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, err
	}
	var plan CleanupPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

// DeletePlan 删除清理计划文件
func DeletePlan(workDir string) error {
	planPath := filepath.Join(workDir, ".theme-cleanup-plan.json")
	return os.Remove(planPath)
}

// sortedKeys 返回 map key 的有序切片
func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// formatValue 格式化配置值用于显示
func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		if len(val) > 60 {
			return fmt.Sprintf("%q...", val[:60])
		}
		return fmt.Sprintf("%q", val)
	case bool:
		return fmt.Sprintf("%v", val)
	case float64:
		return fmt.Sprintf("%v", val)
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}
