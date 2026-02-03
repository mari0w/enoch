---
name: system
description: Detect the current operating system platform (mac/linux/windows) when the user asks about system info.
---

# system

识别当前系统类型，并返回：`mac` / `linux` / `windows` / `unknown`。

## 行为
- 优先使用系统命令或运行环境信息判断。
- 仅输出简洁结论与必要的系统信息。

## 示例
用户：现在是什么系统？
助手：platform: mac
