---
name: os-automation-linux
description: Linux automation toolkit (xdg-open, wmctrl, xdotool, dbus) with safety checks.
---

# os-automation-linux

Linux 自动化指导技能。

## 适用场景
- 打开文件/URL
- 控制窗口焦点、最小化/最大化
- 简单 UI 自动化（X11）

## 优先工具
1. `xdg-open`
2. `wmctrl`（窗口管理）
3. `xdotool`（键鼠自动化）
4. `dbus-send`（桌面环境交互）

## 标准输出模板
- 说明要做的动作
- 给出命令
- 如果需要安装 `wmctrl/xdotool`，先确认

## 常用命令模板
- 打开 URL：`xdg-open "https://example.com"`
- 打开文件：`xdg-open "/path/to/file"`
- 列出窗口：`wmctrl -l`

## 注意事项
- `xdotool` 通常需要 X11；Wayland 可能不可用
- 涉及敏感操作时必须先确认
