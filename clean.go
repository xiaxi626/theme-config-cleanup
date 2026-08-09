package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// runClean 执行 Phase 2 事后清理
func runClean(workDir, themeName string, dryRun bool) error {
	themesDir := filepath.Join(workDir, "themes")
	configPath := filepath.Join(workDir, "config", "config.json")

	mode := "执行清理"
	if dryRun {
		mode = "模拟运行 (--dry-run)"
	}

	fmt.Println("=== 主题配置清理 · 事后清理 ===")
	fmt.Printf("工作区: %s\n", workDir)
	fmt.Printf("已删主题: %s\n", themeName)
	fmt.Printf("模式: %s\n\n", mode)

	// 1. 确认主题已删除
	themePath := filepath.Join(themesDir, themeName)
	themeExists := false
	if _, err := os.Stat(themePath); err == nil {
		if dryRun {
			fmt.Printf("⚠ 主题文件夹仍存在: %s（--dry-run 模式下继续模拟）\n\n", themePath)
			themeExists = true
		} else {
			return fmt.Errorf("主题文件夹仍存在: %s\n请先删除主题文件夹后再运行清理", themePath)
		}
	} else {
		fmt.Println("已确认主题文件夹不存在 ✓")
	}

	// 2. 加载清理计划
	plan, err := LoadPlan(workDir)
	if err != nil {
		return fmt.Errorf("加载清理计划失败: %w\n请先运行 scan 命令生成清理计划", err)
	}
	if plan.ThemeName != themeName {
		return fmt.Errorf("清理计划中的主题名 (%s) 与输入 (%s) 不匹配", plan.ThemeName, themeName)
	}
	fmt.Printf("加载清理计划 ✓（扫描时间: %s）\n", plan.ScanTime)
	fmt.Printf("  - 独占 key: %d 个\n", len(plan.ExclusiveKeys))

	// 3. 二次校验：重新读取剩余主题的 key，防止两阶段间安装了新主题
	// dry-run 模式下若主题仍在，排除目标主题本身避免误判
	excludeName := ""
	if themeExists {
		excludeName = themeName
	}
	currentOtherKeys, _, err := CollectOtherThemeKeys(themesDir, excludeName)
	if err != nil {
		return fmt.Errorf("读取剩余主题 schema 失败: %w", err)
	}

	finalKeysToRemove := make([]string, 0)
	skippedKeys := make([]string, 0)
	for _, key := range plan.ExclusiveKeys {
		if systemKeys[key] {
			skippedKeys = append(skippedKeys, key+" (系统key)")
			continue
		}
		if currentOtherKeys[key] {
			skippedKeys = append(skippedKeys, key+" (新主题已使用)")
			continue
		}
		finalKeysToRemove = append(finalKeysToRemove, key)
	}

	fmt.Printf("  - 二次校验: %d 个可安全删除", len(finalKeysToRemove))
	if len(skippedKeys) > 0 {
		fmt.Printf("，%d 个跳过", len(skippedKeys))
	}
	fmt.Println()

	if len(skippedKeys) > 0 {
		fmt.Println("  跳过的 key:")
		for _, k := range skippedKeys {
			fmt.Printf("    - %s\n", k)
		}
	}

	// 4. 读取当前 customConfig
	config, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("读取 config.json 失败: %w", err)
	}
	customConfig := GetCustomConfig(config)

	// 5. 找出实际存在于 customConfig 中的 key
	keysToRemove := make([]string, 0)
	for _, key := range finalKeysToRemove {
		if _, ok := customConfig[key]; ok {
			keysToRemove = append(keysToRemove, key)
		}
	}

	fmt.Printf("  - config.json 中实际存在: %d 个 key\n\n", len(keysToRemove))

	if len(keysToRemove) == 0 {
		fmt.Println("无需清理的 key，customConfig 中不存在任何独占 key")
		return nil
	}

	// 6. 打印将移除的 key
	fmt.Println("将移除以下 key:")
	for _, key := range keysToRemove {
		fmt.Printf("  - %s = %s\n", key, formatValue(customConfig[key]))
	}

	// 7. 创建清理后的 customConfig 副本，用于图片引用检测
	cleanedConfig := make(map[string]interface{})
	for k, v := range customConfig {
		cleanedConfig[k] = v
	}
	for _, key := range keysToRemove {
		delete(cleanedConfig, key)
	}

	// 8. 图片清理预检
	imageResults, err := checkOrphanedImages(workDir, cleanedConfig)
	if err != nil {
		fmt.Printf("\n⚠ 图片扫描失败: %v\n", err)
	}

	fmt.Println()

	if dryRun {
		fmt.Println("--- 模拟运行完成，未实际修改任何文件 ---")
		fmt.Println("\n如需执行实际清理，去掉 --dry-run 参数重新运行")
		return nil
	}

	// 9. 实际执行
	fmt.Println("--- 执行清理 ---")

	// 备份
	if err := BackupConfig(configPath); err != nil {
		return fmt.Errorf("备份失败: %w", err)
	}
	fmt.Println("已备份: config.json → config.json.bak ✓")

	// 移除 key
	for _, key := range keysToRemove {
		delete(customConfig, key)
	}
	config["customConfig"] = customConfig
	fmt.Printf("已移除 %d 个 key ✓\n", len(keysToRemove))

	// 写回
	if err := SaveConfig(configPath, config); err != nil {
		return fmt.Errorf("写回 config.json 失败: %w（备份在 config.json.bak）", err)
	}
	fmt.Println("已写回 config.json ✓")

	// 图片清理
	if imageResults != nil && len(imageResults.toDelete) > 0 {
		deletedCount, err := deleteOrphanedImages(imageResults.toDelete)
		if err != nil {
			fmt.Printf("⚠ 图片删除部分失败: %v\n", err)
		}
		fmt.Printf("已删除 %d 个孤立图片 ✓\n", deletedCount)
	} else {
		fmt.Println("无孤立图片需要删除")
	}

	// 删除计划文件
	if err := DeletePlan(workDir); err != nil {
		fmt.Printf("⚠ 删除清理计划文件失败: %v\n", err)
	} else {
		fmt.Println("已删除清理计划文件 ✓")
	}

	fmt.Println("\n=== 清理完成 ===")
	return nil
}
