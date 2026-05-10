---
paths:
  - "**/*.go"
  - "**/*_test.go"
---
- subagent 完了直後の `<new-diagnostics>` で `undefined: <symbol>` が並んでも即時 debug や revert に走らない。RED phase テスト由来の遅延通知である可能性が高いため、まず `go test -count=1 ./<pkg>/...` を独立実行し PASS なら diagnostic を stale と判定して reviewer 工程に進む。
