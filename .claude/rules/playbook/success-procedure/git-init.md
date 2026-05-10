---
paths:
  - "**/.gitignore"
  - "**/CLAUDE.md"
  - "**/AGENTS.md"
---
- 新規 repo の first git add 前に project instructions (CLAUDE.md / AGENTS.md) を読み「コミット対象外」と明記された dir と OS メタデータ (.DS_Store 等) を .gitignore に列挙する。git status で untracked から消えたことを確認してから本来のステージへ進む。
