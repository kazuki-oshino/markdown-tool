---
paths:
  - ".claude/skills/**/*.md"
  - ".claude/commands/**/*.md"
  - ".kiro/specs/**"
---
- 計画 / 設計のサニティレビューで subagent を dispatch するときは、親が要約したコンテキストを渡さず、ファイル絶対パスとドラフトのみを渡し subagent 自身に Read させる。親要約を渡すと親バイアスが再生産され独立検証にならない。
