---
paths:
  - "**/*.go"
---
- Go で同一 dir temp + `os.Rename` のアトミック上書きで既存 mode を保持する場合、`os.Chmod` は temp ファイルにのみ適用し destination は決して直接 chmod しない。失敗時の元ファイル不変は defer + success フラグで全早期リターンから `os.Remove(tmp)` を保証する。
