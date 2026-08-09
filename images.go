package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ImageCheckResult 图片清理预检结果
type ImageCheckResult struct {
	totalFiles int
	referenced []string
	toDelete   []string
}

// checkOrphanedImages 扫描 images/theme/ 目录，检查哪些图片未被 customConfig 引用。
// 将整个 customConfig 序列化为 JSON 字符串后做子串匹配，
// 能覆盖字符串值、数组内嵌套对象等各种值类型中的图片路径。
func checkOrphanedImages(workDir string, cleanedConfig map[string]interface{}) (*ImageCheckResult, error) {
	imagesDir := filepath.Join(workDir, "images", "theme")

	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &ImageCheckResult{totalFiles: 0}, nil
		}
		return nil, err
	}

	// 序列化清理后的 customConfig 为 JSON 字符串
	configJSON, _ := json.Marshal(cleanedConfig)
	configStr := string(configJSON)

	result := &ImageCheckResult{totalFiles: len(entries)}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		imagePath := "/images/theme/" + filename

		if strings.Contains(configStr, imagePath) {
			result.referenced = append(result.referenced, filename)
		} else {
			result.toDelete = append(result.toDelete, filepath.Join(imagesDir, filename))
		}
	}

	// 打印报告
	fmt.Printf("\n图片清理 (扫描 images/theme/):\n")
	fmt.Printf("  - 检测到 %d 个文件\n", result.totalFiles)
	fmt.Printf("  - 仍被引用: %d 个\n", len(result.referenced))
	fmt.Printf("  - 将删除: %d 个\n", len(result.toDelete))
	for _, path := range result.toDelete {
		fmt.Printf("    - %s\n", filepath.ToSlash(path))
	}

	return result, nil
}

// deleteOrphanedImages 删除指定的孤立图片文件
func deleteOrphanedImages(paths []string) (int, error) {
	count := 0
	for _, path := range paths {
		if err := os.Remove(path); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}
