---
paths:
  - "**/*.go"
  - "**/*.md"
  - "**/*.ts"
---
- 新規ファイルしか追加しない task で reviewer subagent に `git diff` で変更を見せたい場合、`git add -N <path>` (intent-to-add) で register する。content は staging されないまま `git diff` / `git diff --stat` / `git diff --name-only` に登場し、reviewer 渡し後も通常の `git add <path>` で commit でき併存可能。
