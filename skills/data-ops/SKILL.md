---
name: data-ops
description: Data operations using jq/csvkit/python standard library with minimal dependencies.
---

# data-ops

数据处理指导技能，优先最小依赖。

## 适用场景
- JSON/CSV 过滤、合并、抽取
- 简单统计/清洗

## 优先工具
- JSON：`jq`
- CSV：`csvkit` 或 Python 标准库
- 文本处理：`awk`/`sed`

## 决策模板
- 数据格式？体量？
- 是否已有工具？
- 能否用标准库一次性脚本完成？

## 常见命令模板
- JSON 过滤：`jq '.items[] | {id, name}' input.json`
- CSV 预览：`csvlook data.csv`
- CSV 选择列：`csvcut -c 1,3 data.csv`

## 失败回退
- 先输出一个简化版脚本
- 或提供手工步骤
