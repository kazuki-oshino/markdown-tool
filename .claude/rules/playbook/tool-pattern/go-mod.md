---
paths:
  - "**/*.go"
  - "**/go.mod"
  - "**/go.sum"
---
- Go 外部パッケージ (特に pseudo-version `v0.0.0-<ts>-<hash>`) のソースを参照するときは `find` で広域探索せず、`go env GOMODCACHE` で cache root を取得して `ls $GOMODCACHE/<module-path>@<version>/` で直接開く。dir 名は `<name>@<version>` 形式のため `find -name <name>` 単体では空振りする。
