package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "scan":
		cmdScan(os.Args[2:])
	case "clean":
		cmdClean(os.Args[2:])
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Gridea Pro 主题配置清理工具

用法:
  go run . scan  --dir <工作区路径> --theme <主题文件夹名>
  go run . clean --dir <工作区路径> --theme <主题文件夹名> [--dry-run]

命令:
  scan    预扫描：分析待删主题的独占配置项，生成清理计划
  clean   事后清理：删除主题后，清理残留的 customConfig 和孤立图片

参数:
  --dir      Gridea Pro 工作区路径（包含 config/ 和 themes/ 的目录）
  --theme    待清理的主题文件夹名
  --dry-run  仅在 clean 命令中使用，模拟运行不实际修改文件

示例:
  go run . scan  --dir "C:\Users\user\Documents\Gridea Pro" --theme simple
  go run . clean --dir "C:\Users\user\Documents\Gridea Pro" --theme simple --dry-run
  go run . clean --dir "C:\Users\user\Documents\Gridea Pro" --theme simple`)
}

func cmdScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	dir := fs.String("dir", "", "Gridea Pro 工作区路径")
	theme := fs.String("theme", "", "待删主题文件夹名")
	fs.Parse(args)

	if *dir == "" || *theme == "" {
		fmt.Fprintln(os.Stderr, "错误: --dir 和 --theme 参数必填")
		fs.Usage()
		os.Exit(1)
	}

	if err := runScan(*dir, *theme); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}

func cmdClean(args []string) {
	fs := flag.NewFlagSet("clean", flag.ExitOnError)
	dir := fs.String("dir", "", "Gridea Pro 工作区路径")
	theme := fs.String("theme", "", "已删主题文件夹名")
	dryRun := fs.Bool("dry-run", false, "模拟运行，不实际修改文件")
	fs.Parse(args)

	if *dir == "" || *theme == "" {
		fmt.Fprintln(os.Stderr, "错误: --dir 和 --theme 参数必填")
		fs.Usage()
		os.Exit(1)
	}

	if err := runClean(*dir, *theme, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
}
