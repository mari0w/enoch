---
name: codex-skills
description: Ensure Codex only uses repo-local skills by setting CODEX_HOME to the project root.
---

# codex-skills

规范 Codex 在项目内读取技能的方式，避免使用全局技能。

## 关键点
- 优先使用项目内 `skills/`。
- 通过仓库内 `.codex/skills` 引导 Codex 只加载项目技能（本项目使用符号链接）。
- 不建议覆盖 `CODEX_HOME`，避免影响已有登录缓存。

## 建议
- 维护 `skills/` 为单一来源，并让 `.codex/skills -> skills/`。
- 仅在需要共享时才使用全局技能目录。
