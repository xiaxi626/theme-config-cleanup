package main

import (
	"fmt"
	"path/filepath"
)

// runScan 执行 Phase 1 预扫描
func runScan(workDir, themeName string) error {
	themesDir := filepath.Join(workDir, "themes")
	configPath := filepath.Join(workDir, "config", "config.json")

	fmt.Println("=== 主题配置清理 · 预扫描 ===")
	fmt.Printf("工作区: %s\n", workDir)
	fmt.Printf("待删主题: %s\n\n", themeName)

	// 1. 验证待删主题存在
	targetSchema, err := LoadThemeSchema(themesDir, themeName)
	if err != nil {
		return fmt.Errorf("无法读取待删主题 schema（确认主题文件夹存在且包含 config.json）: %w", err)
	}

	targetKeys := ExtractThemeKeys(targetSchema)
	fmt.Printf("待删主题: %s v%s（%d 个配置项）\n", targetSchema.Name, targetSchema.Version, len(targetKeys))

	// 2. 读取其他主题 schema
	otherKeys, themeNames, err := CollectOtherThemeKeys(themesDir, themeName)
	if err != nil {
		return fmt.Errorf("读取其他主题 schema 失败: %w", err)
	}

	if len(themeNames) > 0 {
		fmt.Println("其他已安装主题:")
		for _, name := range themeNames {
			schema, _ := LoadThemeSchema(themesDir, name)
			if schema != nil {
				fmt.Printf("  - %s v%s（%d 个配置项）\n", schema.Name, schema.Version, len(ExtractThemeKeys(schema)))
			}
		}
	}

	// 3. 计算独占 key 和共享 key
	exclusiveKeys := make(map[string]bool)
	sharedKeys := make(map[string]bool)
	for k := range targetKeys {
		if otherKeys[k] {
			sharedKeys[k] = true
		} else {
			exclusiveKeys[k] = true
		}
	}

	// 4. 读取当前 customConfig
	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("读取 config.json 失败: %w", err)
	}
	customConfig := GetCustomConfig(config)

	// 5. 记录独占 key 在 customConfig 中的实际值
	currentValues := make(map[string]interface{})
	for k := range exclusiveKeys {
		if v, ok := customConfig[k]; ok {
			currentValues[k] = v
		}
	}

	// 6. 打印报告
	fmt.Println()
	fmt.Printf("独占 key（仅 %s 使用，将被清理）: %d 个\n", themeName, len(exclusiveKeys))
	if len(exclusiveKeys) == 0 {
		fmt.Println("  （无）")
	} else {
		for _, k := range sortedKeys(exclusiveKeys) {
			if v, ok := currentValues[k]; ok {
				fmt.Printf("  - %s = %s\n", k, formatValue(v))
			} else {
				fmt.Printf("  - %s（未在 customConfig 中设置）\n", k)
			}
		}
	}

	fmt.Println()
	fmt.Printf("共享 key（其他主题也在使用，将保留）: %d 个\n", len(sharedKeys))
	if len(sharedKeys) == 0 {
		fmt.Println("  （无）")
	} else {
		for _, k := range sortedKeys(sharedKeys) {
			fmt.Printf("  - %s\n", k)
		}
	}

	fmt.Println()
	fmt.Printf("系统注入 key（将保留）: %d 个\n", len(systemKeys))
	for _, k := range sortedKeys(systemKeys) {
		fmt.Printf("  - %s\n", k)
	}

	// 7. 保存清理计划
	plan := &CleanupPlan{
		ThemeName:     themeName,
		ExclusiveKeys: sortedKeys(exclusiveKeys),
		SharedKeys:    sortedKeys(sharedKeys),
		CurrentValues: currentValues,
	}
	if err := SavePlan(workDir, plan); err != nil {
		return fmt.Errorf("保存清理计划失败: %w", err)
	}

	fmt.Println()
	fmt.Println("清理计划已保存: .theme-cleanup-plan.json")
	fmt.Printf("\n✓ 现在可以安全删除 themes/%s 文件夹了\n", themeName)
	fmt.Printf("  删除后运行: go run . clean --dir \"%s\" --theme %s\n", workDir, themeName)

	return nil
}
