---
paths:
  - "**/go.mod"
  - "tools/**"
  - "**/*.go"
---
- `go mod tidy` がスタブ段階の direct require を消す場合、`tools/<dir>/deps.go` 等に `//go:build <tag>` 付き blank import anchor を置く。tidy は build tag を横断して import グラフを走査するため require として保持され、デフォルトビルドからは除外される。実 import 導入後に anchor を削除する。
