---
name: assistant-playbook
description: Problem-solving workflow: check OS, check skills, reuse tools, and install missing tools safely.
---

# assistant-playbook

这是项目内的“解决问题流程”规范。用于在聊天中接到任务时，优先复用现有技能与系统能力，必要时再安装工具或编写脚本。

## 核心原则
1. 先判定系统类型与版本（mac/linux/windows）再做方案选择。
2. 优先使用项目内现有 `skills/` 能力；没有就评估系统自带能力。
3. 若缺少关键能力：优先选择成熟工具/库；安装前必须说明原因并征得确认。
4. 尽量用最小依赖完成任务；可脚本化就脚本化，避免引入重依赖。
5. 失败要有回退方案（替代工具/人工步骤）。

## 决策流程（细化）
- Step 1: 使用 `system` 技能判断 OS/版本。
- Step 2: 查 `skills/` 是否已有对应能力。
- Step 3: 如果无技能，评估系统原生工具：
  - macOS: `osascript`/Shortcuts/Automator
  - Linux: `xdg-open`/`dbus`/`wmctrl`/`xdotool`
- Step 4: 若仍不够，用最小依赖方案：
  - 文档类：`pandoc`/`python-docx`/`libreoffice --headless`
  - 数据类：`csvkit`/`jq`/`python` 标准库
  - 自动化类：`bash`/`zsh`/`expect`
- Step 5: 需要安装时说明：用途、影响、替代方案，并请求用户确认。

## 输出模板
- 我将先检查当前系统类型。
- 我会优先使用现有技能/系统工具。
- 如果必须安装工具，会先说明原因并征求同意。

## 安装策略（仅 mac/linux）
- macOS 优先 `brew install <pkg>`
- Linux 按发行版选择：`apt` / `dnf` / `yum` / `pacman`

## 失败回退
- 给出替代工具/手动步骤
- 或提供简化版脚本
