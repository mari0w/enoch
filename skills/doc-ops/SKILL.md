---
name: doc-ops
description: Document operations (docx/pdf/markdown) using minimal tools or libraries.
---

# doc-ops

文档操作指导技能，优先最小依赖完成。

## 适用场景
- 生成/编辑 Markdown、DOCX、PDF
- 格式转换（md <-> docx/pdf）

## 优先方案
1. 纯文本/Markdown：直接生成或编辑
2. DOCX：`pandoc` 或 `python-docx`
3. PDF：`pandoc` 或 `libreoffice --headless`

## 决策模板
- 目标格式？
- 是否需要保持原格式？
- 系统是否已有工具？
- 如需安装，先说明并征求同意

## 常见命令模板
- Markdown -> PDF：`pandoc input.md -o output.pdf`
- Markdown -> DOCX：`pandoc input.md -o output.docx`
- DOCX -> PDF：`libreoffice --headless --convert-to pdf input.docx`

## 失败回退
- 提供手动导出步骤
- 或改用 Markdown 交付
