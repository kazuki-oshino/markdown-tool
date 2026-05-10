---
paths:
  - "cmd/**/*.go"
  - "Justfile"
---
- CLI バイナリの smoke 検証は `--help` exit 0 で停止せず、`mktemp` 等で生成した実入力に対して実サブコマンドを実行し stdout / 上書き結果 / exit code の三点を観測してから PASS にする。`--help` は引数解析パスしか触らず、依存パッケージ初期化や I/O 副作用層の致命傷を取りこぼす。
