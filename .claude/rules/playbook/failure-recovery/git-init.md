---
paths:
  - "**/.gitignore"
  - "**/CLAUDE.md"
  - "**/AGENTS.md"
---
- 新規 init 直後の repo で `git log` が exit 128 / 'your current branch does not have any commits yet' を返したら repo 破損ではなく commit 0 件 の signal。`git status` の 'No commits yet' と併せて初期 commit 経路に進み、retry や再 init はしない。
