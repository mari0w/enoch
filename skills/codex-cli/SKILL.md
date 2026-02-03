---
name: codex-cli
description: Use Codex CLI in non-interactive mode (codex exec) for automation and bots.
---

# codex-cli

用于通过 Codex CLI 进行非交互式调用的规范。

## 核心要点
- `codex` 不带子命令会启动交互式 TUI，需要真实终端。
- 自动化/脚本场景应使用 `codex exec`，它是非交互式模式，输出结果到 stdout。

## 推荐用法
- 通过参数传入 prompt：
  - `codex exec "<prompt>"`
- 需要稳定自动化时，避免启动交互式 TUI。

## 适用场景
- 机器人轮询消息 -> 调用 Codex CLI -> 返回输出。
