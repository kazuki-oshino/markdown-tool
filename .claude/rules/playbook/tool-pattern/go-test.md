---
paths:
  - "**/*.go"
  - "**/*_test.go"
  - "Justfile"
---
- 完了主張の根拠に使う Go テストは `go test -count=1 ./...` で test cache を bust する。`(cached)` 付き出力は前回 build 時点の証跡で、最新コードに対する PASS evidence として不十分。`just test` 等が `-count=1` を持たないなら raw コマンドを直接呼ぶ。
- `os.Chmod(dir, 0o555)` で書き込み失敗を再現するテストは先頭で `if os.Geteuid() == 0 { t.Skip("root では chmod 制限が効かない") }` を置く。Docker / CI / sudo 実行では chmod が無効化され書き込みが成功し、失敗パステストが false negative になる。Windows も同様に skip する。
