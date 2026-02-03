#!/usr/bin/env python3
from __future__ import annotations

import argparse
import datetime as dt
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MEMORY_DIR = ROOT / "memory"
TEMPLATE_PATH = ROOT / "skills" / "memory" / "MEMORY_TEMPLATE.md"

DEFAULT_TEMPLATE = """# {{date}}

## Summary
- 

## Decisions
- 

## TODOs
- 

## Context
- 

## Prompts/Rules
- 
"""


def load_template() -> str:
    if TEMPLATE_PATH.exists():
        text = TEMPLATE_PATH.read_text(encoding="utf-8")
        match = re.search(
            r"<!-- TEMPLATE START -->\n(.*?)\n<!-- TEMPLATE END -->",
            text,
            re.S,
        )
        if match:
            return match.group(1)
    return DEFAULT_TEMPLATE


def today_str() -> str:
    return dt.date.today().strftime("%Y-%m-%d")


def ensure_today_file() -> Path:
    MEMORY_DIR.mkdir(parents=True, exist_ok=True)
    path = MEMORY_DIR / f"{today_str()}.md"
    if not path.exists():
        template = load_template().replace("{{date}}", today_str()).rstrip() + "\n"
        path.write_text(template, encoding="utf-8")
        print(f"Created {path}")
    return path


def insert_into_context(text: str, entry: str) -> str:
    lines = text.splitlines()
    header = "## Context"
    for i, line in enumerate(lines):
        if line.strip() == header:
            insert_at = i + 1
            while insert_at < len(lines) and lines[insert_at].strip() == "":
                insert_at += 1
            lines.insert(insert_at, entry)
            return "\n".join(lines).rstrip() + "\n"
    return text.rstrip() + "\n\n" + entry + "\n"


def add_entry(message: str) -> None:
    path = ensure_today_file()
    timestamp = dt.datetime.now().strftime("%H:%M")
    entry = f"- [{timestamp}] {message}"
    content = path.read_text(encoding="utf-8")
    updated = insert_into_context(content, entry)
    path.write_text(updated, encoding="utf-8")
    print(f"Appended to {path}")


def search(keyword: str) -> int:
    if not MEMORY_DIR.exists():
        print("No memory directory found.")
        return 1
    needle = keyword.lower()
    found = False
    for path in sorted(MEMORY_DIR.glob("*.md")):
        try:
            lines = path.read_text(encoding="utf-8").splitlines()
        except OSError:
            continue
        for idx, line in enumerate(lines, start=1):
            if needle in line.lower():
                print(f"{path.name}:{idx}: {line}")
                found = True
    return 0 if found else 1


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Memory helper")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("init", help="Create today's memory file if missing")

    add_parser = sub.add_parser("add", help="Append an entry with timestamp")
    add_parser.add_argument("message", help="Entry text")

    search_parser = sub.add_parser("search", help="Search memory files by keyword")
    search_parser.add_argument("keyword", help="Keyword to search")

    return parser


def main() -> int:
    parser = build_parser()
    args = parser.parse_args()

    if args.command == "init":
        ensure_today_file()
        return 0
    if args.command == "add":
        add_entry(args.message)
        return 0
    if args.command == "search":
        return search(args.keyword)

    parser.print_help()
    return 1


if __name__ == "__main__":
    sys.exit(main())
