# 開発スタイル

TDD で開発する（探索 → Red → Green → Refactoring）。
KPI やカバレッジ目標が与えられたら、達成するまで試行する。
不明瞭な指示は質問して明確にする。

# コード設計

- 関心の分離を保つ
- 状態とロジックを分離する
- 可読性と保守性を重視する
- コントラクト層（API/型）を厳密に定義し、実装層は再生成可能に保つ
- 静的検査可能なルールはプロンプトではなく、その環境の linter か ast-grep で記述する

# ツール

- タスク: just
- Go: v1.25+

# 破壊的操作

- ツール（home-manager / brew / chezmoi / pre-commit / pip / npm 等）が auto-rename した `*.backup` / `*.orig` / `*.pre-*` 系を `rm` する前に、内容を `cat` して会話に出すか別ファイルに dump する。最低 1 回の表示を経てから削除する
  （理由: 自分が作ったファイルではないので、消すと「元に何が入っていたか」が永久に失われる。`/etc/zshenv` のような system-level 置き土産が紛れていても気づけなくなる）

# 自己学習アーティファクトの保全

memory-distiller は、akashic ログから抽出した失敗パターン・修復ノウハウを `.claude/rules/playbook/<category>/<topic>.md` に追記する。これらは AI が過去セッションで蓄えた「自己学習成果物」であり、人間ユーザは通常編集しない。AI が以下の操作を行うときは、これらを「身に覚えのない差分」として扱ってはならない:

- **コミットを巻き戻す**（`git revert` / `git reset` / `git restore` / `git checkout --` / amend での除外）: `.claude/rules/playbook/` 配下に対するコミットは memory-distiller による正規の自己学習であると仮定する。ユーザーが明示的にそのコミットの巻き戻しを指示した場合のみ実行する
- **ファイルを削除する**（`rm` / `git clean -f` / Edit で空にする / Write で全置換）: `.claude/rules/playbook/` 配下のファイルは、ユーザーの明示指示なしに削除・全置換しない
- **コミットを作成する**: 自分が当該セッションで `.claude/rules/playbook/` 配下を編集していない場合、その差分は別エージェント（memory-distiller の apply phase）が直前に書いた学習成果物の可能性が高いので、`git add -A` で巻き込まずに別コミットへ分離するか、ステージから外す

例外: ユーザーから明示的に「memory-distiller の出力をリセットする」「playbook を初期化する」等の指示がある場合は、上記「# 破壊的操作」のプロトコル（内容を `cat` で表示してから削除）に従う。

`.rill/memory-distiller/<runID>/*.json` は一時的な中間アーティファクトなので削除してよい（コミット対象外として `.gitignore` するのが望ましい）。

# スキル作成

新規 skill を作るとき、配置先を次の指針で決める:

- **project 固有** (`<repo>/.claude/skills/` に置く / 該当 repo の `apm.yml` で配布): 特定 repo のドメイン知識・規約・ファイルレイアウトに依存し、他 repo で使う見込みがない
- **グローバル** (`~/.claude/skills/` 直置き or APM global): 言語・ツール横断、複数 repo で再利用可能、運用ノウハウ
- **判断不能なとき**: ユーザーに「project 固有かグローバルか」を質問してから作成（理由: 後から移動するとパス参照や apm.yml 設定が壊れやすい）


# Agentic SDLC and Spec-Driven Development

Kiro-style Spec-Driven Development on an agentic SDLC

## Project Context

### Paths
- Steering: `.kiro/steering/`
- Specs: `.kiro/specs/`

### Steering vs Specification

**Steering** (`.kiro/steering/`) - Guide AI with project-wide rules and context
**Specs** (`.kiro/specs/`) - Formalize development process for individual features

### Active Specifications
- Check `.kiro/specs/` for active specifications
- Use `/kiro-spec-status [feature-name]` to check progress

## Development Guidelines
- Think in English, generate responses in Japanese. All Markdown content written to project files (e.g., requirements.md, design.md, tasks.md, research.md, validation reports) MUST be written in the target language configured for this specification (see spec.json.language).

## Minimal Workflow
- Phase 0 (optional): `/kiro-steering`, `/kiro-steering-custom`
- Discovery: `/kiro-discovery "idea"` — determines action path, writes brief.md + roadmap.md for multi-spec projects
- Phase 1 (Specification):
  - Single spec: `/kiro-spec-quick {feature} [--auto]` or step by step:
    - `/kiro-spec-init "description"`
    - `/kiro-spec-requirements {feature}`
    - `/kiro-validate-gap {feature}` (optional: for existing codebase)
    - `/kiro-spec-design {feature} [-y]`
    - `/kiro-validate-design {feature}` (optional: design review)
    - `/kiro-spec-tasks {feature} [-y]`
  - Multi-spec: `/kiro-spec-batch` — creates all specs from roadmap.md in parallel by dependency wave
- Phase 2 (Implementation): `/kiro-impl {feature} [tasks]`
  - Without task numbers: autonomous mode (subagent per task + independent review + final validation)
  - With task numbers: manual mode (selected tasks in main context, still reviewer-gated before completion)
  - `/kiro-validate-impl {feature}` (standalone re-validation)
- Progress check: `/kiro-spec-status {feature}` (use anytime)

## Skills Structure
Skills are located in `.claude/skills/kiro-*/SKILL.md`
- Each skill is a directory with a `SKILL.md` file
- Skills run inline with access to conversation context
- Skills may delegate parallel research to subagents for efficiency
- Additional files (templates, examples) can be added to skill directories
- `kiro-review` — task-local adversarial review protocol used by reviewer subagents
- `kiro-debug` — root-cause-first debug protocol used by debugger subagents
- `kiro-verify-completion` — fresh-evidence gate before success or completion claims
- **If there is even a 1% chance a skill applies to the current task, invoke it.** Do not skip skills because the task seems simple.

## Development Rules
- 3-phase approval workflow: Requirements → Design → Tasks → Implementation
- Human review required each phase; use `-y` only for intentional fast-track
- Keep steering current and verify alignment with `/kiro-spec-status`
- Follow the user's instructions precisely, and within that scope act autonomously: gather the necessary context and complete the requested work end-to-end in this run, asking questions only when essential information is missing or the instructions are critically ambiguous.

## Steering Configuration
- Load entire `.kiro/steering/` as project memory
- Default files: `product.md`, `tech.md`, `structure.md`
- Custom files are supported (managed via `/kiro-steering-custom`)
