---
paths:
  - "internal/**/*.go"
  - "**/*_test.go"
  - "testdata/**"
---
- 新規公開 API + golden test の RED phase で「movement あり fixture のみ失敗」を作るには、最初に identity stub (`return input, nil`) で実装してから golden 全件を流す。no-op fixture は pass し movement fixture だけが mismatch するため、テストが本当に movement を見ているかを構造的に確認できる。
