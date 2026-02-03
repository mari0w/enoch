---
name: os-automation-mac
description: macOS automation toolkit (osascript/Shortcuts/Automator) and safe command usage.
---

# os-automation-mac

macOS 自动化指导技能。

## 适用场景
- 打开/切换应用、窗口
- 打开文件/URL
- 系统状态查询（非敏感）

## 优先工具
1. AppleScript（`osascript`）
2. Shortcuts（`shortcuts run <name>`）
3. Automator（仅在无其他方案时）

## 标准输出模板
- 说明要做的动作
- 给出命令
- 如果会影响系统/数据，先确认

## 常用命令模板
- 打开应用：`osascript -e 'tell application "Finder" to activate'`
- 打开 URL：`open "https://example.com"`
- 打开文件：`open "/path/to/file"`
- 切换前台应用：`osascript -e 'tell application "Safari" to activate'`

## 注意事项
- 自动化权限（Accessibility/Automation）可能需要用户手动授权
- 涉及隐私/系统设置时必须先确认
