---
name: tooling-installer
description: Install missing tools safely (mac: brew, linux: apt/dnf/yum/pacman). Always ask for confirmation.
---

# tooling-installer

用于在缺少工具时进行“安全安装”的指导技能。

## 适用场景
- 现有技能与系统自带工具无法满足需求。
- 需要新增命令/包/运行时才能完成任务。

## 固定话术模板（必须先征得同意）
- 说明：为什么需要安装
- 影响：安装什么包、体积/时间预估
- 备选：不安装的替代方案（若有）
- 询问：是否允许安装

示例：
- 需要安装 `pandoc` 才能把 Markdown 转成 PDF。
- 预计下载约 50-100MB。
- 备选方案是手动导出。
- 是否允许我安装？

## 平台安装策略
- macOS: `brew install <pkg>`
- Debian/Ubuntu: `sudo apt-get update && sudo apt-get install -y <pkg>`
- RHEL/CentOS: `sudo yum install -y <pkg>` 或 `sudo dnf install -y <pkg>`
- Arch: `sudo pacman -S --noconfirm <pkg>`

## 安装后验证
- 给出验证命令：`<pkg> --version`
- 如果失败，给出回退方案

## 常见工具映射
- 文档: `pandoc`, `libreoffice` (headless)
- 自动化: `xdotool`, `wmctrl`
- 数据: `jq`, `csvkit`
