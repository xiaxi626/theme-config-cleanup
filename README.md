# Gridea Pro 主题配置清理工具

清理已删除主题残留在 Gridea Pro `config/config.json` 中的 `customConfig` 配置项和孤立图片。

## 背景

Gridea Pro 的所有主题共用 `config/config.json` 中的同一个 `customConfig` 字段。删除主题文件夹后，该主题的配置项会残留在 `customConfig` 中，不会被自动清理。本工具通过两阶段扫描精确识别并清理这些残留数据。

## 前置条件

- 已安装 Go 1.21+
- Gridea Pro 工作区路径（包含 `config/` 和 `themes/` 的目录）

## 使用方式

### Phase 1：预扫描（删除主题前）

```bash
go run . scan --dir "C:\Users\用户名\Documents\Gridea Pro" --theme simple
```

程序会：
1. 读取待删主题的 schema，提取其所有配置项 key
2. 读取其他已安装主题的 schema，收集它们的 key
3. 计算**独占 key**（仅待删主题使用）和**共享 key**（其他主题也在用）
4. 生成清理计划文件 `.theme-cleanup-plan.json`
5. 输出报告，告知哪些 key 将被清理、哪些将保留

扫描完成后，即可安全删除主题文件夹。

### Phase 2：事后清理（删除主题后）

```bash
# 先用 --dry-run 模拟运行，确认无误
go run . clean --dir "C:\Users\用户名\Documents\Gridea Pro" --theme simple --dry-run

# 确认无误后，执行实际清理
go run . clean --dir "C:\Users\用户名\Documents\Gridea Pro" --theme simple
```

程序会：
1. 确认主题文件夹已删除（`--dry-run` 模式下允许主题仍存在，仅警告）
2. 加载 Phase 1 生成的清理计划
3. **二次校验**：重新读取剩余主题的 key，防止两阶段间安装了新主题碰巧使用同名 key
4. 备份 `config.json` → `config.json.bak`（dry-run 跳过）
5. 从 `customConfig` 中移除独占 key（dry-run 跳过）
6. 扫描 `images/theme/` 目录，删除未被清理后 `customConfig` 引用的孤立图片（dry-run 跳过）
7. 删除清理计划文件（dry-run 跳过）

> **提示**：`--dry-run` 可在主题文件夹仍存在时运行，用于 scan 后、删除前的测试预览。

## 参数说明

| 参数 | 说明 | 必填 |
|------|------|------|
| `--dir` | Gridea Pro 工作区路径 | 是 |
| `--theme` | 主题文件夹名 | 是 |
| `--dry-run` | 模拟运行，不实际修改文件（仅 clean 命令） | 否 |

## 安全机制

- **备份**：清理前自动备份 `config.json` 到 `config.json.bak`
- **dry-run**：可先模拟运行确认清理内容，不修改任何文件；可在主题删除前用于测试预览
- **二次校验**：Phase 2 会重新扫描剩余主题，若两阶段间安装了使用同名 key 的新主题，该 key 会被跳过
- **系统 key 保护**：引擎注入的 `avatar`、`links` 等 key 不会被清理
- **共享 key 保护**：其他主题仍在使用的 key 不会被清理

## 工作原理

```
Phase 1 (主题仍在)                Phase 2 (主题已删)
┌──────────────────────┐          ┌──────────────────────┐
│ 读取待删主题 schema   │          │ 确认主题文件夹已消失   │
│ 读取其他主题 schema   │    →     │ 加载清理计划           │
│ 计算 exclusiveKeys   │  用户     │ 二次校验 key 安全性    │
│ 生成清理计划文件      │  删除     │ 清理 customConfig     │
│ 告知用户可安全删除    │  主题     │ 清理孤立图片           │
└──────────────────────┘          └──────────────────────┘
```

**核心公式**：`exclusiveKeys = 待删主题的 key 集合 − 所有其他主题的 key 集合`

只有独占 key 会被清理；与其他主题共享的 key 会保留。

## 文件结构

```
theme-config-cleanup/
├── go.mod        Go 模块定义
├── main.go       CLI 入口与子命令路由
├── config.go     配置/schema/计划文件的读写
├── scan.go       Phase 1 预扫描逻辑
├── clean.go      Phase 2 事后清理逻辑
├── images.go     孤立图片检测与清理
└── README.md     本文件
```

## 注意事项

- 建议在 Gridea Pro 未运行时执行清理，避免文件写入冲突
- 清理计划文件 `.theme-cleanup-plan.json` 保存在工作区根目录，Phase 2 完成后自动删除
- 如果计划文件丢失，Phase 2 将无法执行（需重新运行 Phase 1）
- 图片清理采用全目录扫描方式：将清理后的 `customConfig` 序列化为 JSON 字符串，检查 `images/theme/` 下每个文件的路径是否被引用，未引用则删除
