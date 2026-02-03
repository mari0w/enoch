---
name: memory
description: Read/write project memory files and search past context.
---

# memory

用于项目内“记忆”文件的读写与检索。

## When to load
- 默认加载今天 + 昨天的记忆文件（`memory/YYYY-MM-DD.md`）。
- 用户显式要求“查历史/回忆/之前做过什么”时，再按关键词搜索全部记忆文件。
- 需要上下文但日期不清晰时，先问日期范围或关键词。

## How to read
- 读取 `memory/YYYY-MM-DD.md`，聚焦 Summary/Decisions/TODOs/Context/Prompts/Rules。
- 不要一次性粘贴整文件，优先摘要。

## How to write
- 追加简短条目，优先放在对应分区；带时间戳（HH:MM）。
- 新建当天文件时使用模板：`skills/memory/MEMORY_TEMPLATE.md`。

## Helpers
- 使用脚本：`python3 scripts/memory.py init|add|search`。
