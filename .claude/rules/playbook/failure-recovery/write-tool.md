---
paths:
  - "**/*.go"
  - "**/*.md"
  - "**/*.json"
---
- Write で既存ファイルを全置換する前に必ず同一 session 内で Read を呼ぶ。'File has not been read yet' で失敗したら Read → 同じ内容で Write 再実行で復旧する (Edit へ切替えない)。
