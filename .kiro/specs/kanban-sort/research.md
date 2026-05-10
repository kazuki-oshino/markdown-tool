# Gap Analysis: kanban-sort

> 言語: ja（spec.json に従う）
> 対象 spec: `.kiro/specs/kanban-sort/`
> 前提: グリーンフィールド（Go モジュール未初期化・既存コードなし）

## 0. 注意事項 / 前提

- **`.kiro/steering/` が未存在**: `product.md` / `tech.md` / `structure.md` がリポジトリ上に存在しないため、ステアリング由来の制約は CLAUDE.md と本 brief / requirements に明記された制約のみを採用した。後段で `/kiro-steering` 実施を推奨。
- **既存コード資産なし**: ルートは `.claude/`、`.kiro/`、`.rill/`、`.gitignore`、`CLAUDE.md` のみ。`go.mod`・`Justfile`・`main.go` ともに存在しない。よって本ギャップ解析は「拡張」より「新規構築の構造選択」を主軸に行う。
- **明示済みの上流前提**: Go 1.25+ / Cobra v1.10+ / Bubble Tea v1 系。これらは brief で確定済みの前提条件として扱い、選定の代替案は出さない。

## 1. Current State Investigation

### 1.1 リポジトリ構成スナップショット

| パス | 役割 | 備考 |
| --- | --- | --- |
| `CLAUDE.md` | プロジェクト規約 | TDD・関心の分離・linter / ast-grep 寄せ・`just` 採用 |
| `.kiro/specs/kanban-sort/` | 本 spec | requirements 生成済み・design / tasks 未着手 |
| `.kiro/steering/` | （未作成） | ステアリング不在 |
| `.claude/skills/` | プロジェクト固有 skill | kiro-* 系一式 |
| `.gitignore` | `.DS_Store` / `.rill/` のみ | Go / IDE / バイナリ ignore は未整備 |

### 1.2 ドメイン関連資産の有無

- **マークダウンパーサ**: 既存実装なし。`goldmark` 等の AST 系ライブラリも未導入（brief で明示的に「AST 不要」と決定済み）。
- **CLI 雛形**: 既存実装なし。Cobra も未依存。
- **TUI 雛形**: 既存実装なし。Bubble Tea も未依存。
- **テスト基盤**: 既存実装なし。Go 標準 `testing` 利用想定。
- **タスクランナ**: `Justfile` 未配置（CLAUDE.md で `just` 採用方針が明示されているが、recipe 定義は本 spec 範囲で新規作成が必要）。

### 1.3 既存規約 / 制約（CLAUDE.md 由来）

- TDD（探索 → Red → Green → Refactor）を遵守する。
- 関心の分離・状態とロジックの分離・コントラクト層の厳密化を優先する。
- 静的検査ルールはプロンプトでなく linter / ast-grep に寄せる。
- 破壊的操作前に `*.backup` 等の中身を 1 度は表示する規約あり（CLI 自身の機能に含めない、開発作業上の注意）。
- `.claude/rules/playbook/` 配下は memory-distiller 出力につき編集・削除は明示指示時のみ。

## 2. Requirement-to-Asset Map（ギャップ表）

| Req | 概要 | 既存資産 | ギャップ種別 | 必要となる新規実装 |
| --- | --- | --- | --- | --- |
| R1 | `[x]` ブロックを完了集末尾へ移動 | なし | Missing | 行ベースパーサ + 移動ロジック（純関数） |
| R2 | 完了集／未完了パートの境界判定 | なし | Missing | 「最初の `[ ]` 行直前」を境界とするスキャナ |
| R3 | 親子インデント尊重（Tab / 半角 2 個以上） | なし | Missing | インデント深さ正規化＋親子ブロック判定＋集約移動 |
| R4 | `--dry-run` と即上書き、改行コード保持 | なし | Missing / Unknown | 差分表示フォーマット未確定（要研究）／改行・末尾改行の保持戦略／アトミック書き込み |
| R5 | `sort` サブコマンド・終了コード規約・`--help` | なし | Missing | Cobra ルート＋`sort` サブコマンドの雛形 |
| R6 | 引数なし起動時の TUI スケルトン（`q`/`Ctrl+C` で終了） | なし | Missing | Bubble Tea Model/Update/View の最小実装＋Cobra との接続 |
| R7 | ロジックの純粋性・決定性・I/O 分離 | なし | Constraint（横断要件） | パッケージ分割で I/O ↔ 純ロジックを分離する設計判断が必要 |

### 2.1 機能横断の制約整理

- **純粋性 (R7)**: ソート関数のシグネチャは `func Sort(input string) string`（または `(string, error)` のいずれか）に固定したい。ファイル I/O・diff 出力・TUI はすべてこの純関数の外側で実装する。
- **決定性 (R7-2)**: 同一入力 → 同一出力。乱数・time・並列順序非依存にする。テーブル駆動テスト＋ゴールデンファイルで担保しやすい。
- **副作用境界 (R1-4 / R7-3)**: 対象ファイル以外への書き込み禁止。これはテストでも徹底（`t.TempDir()` 内の対象ファイル以外が書かれていないことを検査）する余地あり。

## 3. 複雑性シグナル（Complexity Signals）

- **アルゴリズム的ロジック**: 親子ブロック判定・移動可否判定は単純な木構造アルゴリズム。再帰または手続き的スキャンで十分。
- **外部統合**: なし（git・ネットワーク・サブプロセスを禁止）。
- **UI**: TUI は最小スケルトン（起動・初期画面・終了のみ）に限定されており、複雑性は低い。
- **データ整合性**: ファイル上書き時の中間破損禁止（R4-5）→ アトミック書き込み（temp file + `os.Rename`）が必要。
- **エンコーディング**: 改行コード（LF/CRLF）と末尾改行の保持（R4-4）→ 入力時の検出と出力時の再付与ロジックが必要。

## 4. Implementation Approach Options

> 本プロジェクトは greenfield のため「既存コンポーネントの拡張」は適用不能。代わりに **どの粒度でパッケージを切るか** を選択肢化する。

### Option A: フラット 2 層（`cmd/mdt` + `internal/sort` のみ）

**いつ採用するか**: 初期スコープが「`sort` 1 サブコマンド + TUI スケルトン」に閉じる前提で、立ち上がりを最速にしたい場合。

- **構成例**:
  - `cmd/mdt/main.go`: Cobra ルート定義のエントリポイント
  - `internal/sort/`: 純関数 `Sort(string) string`、行パーサ、ブロック判定をすべて 1 パッケージに収める
  - `internal/tui/`: Bubble Tea スケルトン（任意で `internal/sort` から分離）
- **トレードオフ**:
  - ✅ ファイル数が最少で TDD の初動が速い
  - ✅ R7 の純粋性は `internal/sort` を CLI / TUI から呼ぶ形で守れる
  - ❌ 将来 `mdt format` 等が増えた際、パーサや diff ヘルパが `internal/sort` に閉じ込められて再利用しづらい
  - ❌ パーサ / 境界判定 / 親子判定 / 再構築 が同居し、責務境界が曖昧になりやすい

### Option B: 4–5 層クリーン構成（パーサ / ドメイン / I/O / CLI / TUI を分離）

**いつ採用するか**: brief の「Boundary Candidates」で示された 5 レイヤをそのまま反映し、将来サブコマンドからの再利用を前提にしたい場合。

- **構成例**:
  - `internal/markdown/`: 行 → トークン（インデント深さ・チェック状態・残テキスト）への純パーサ
  - `internal/kanban/`: 完了集／未完了境界・親子ツリー・移動可否判定・再構築（純ドメインロジック）
  - `internal/fileio/`: 読み書き・改行コード保持・アトミック上書き・dry-run の差分生成
  - `internal/cliapp/` または `cmd/mdt/`: Cobra サブコマンド・フラグ・終了コード規約
  - `internal/tui/`: Bubble Tea スケルトン
- **トレードオフ**:
  - ✅ 境界が明示され、TDD のスコープを各層単位で絞れる（純パーサ・純ドメインの両方を独立にゴールデンテスト可能）
  - ✅ 将来 `mdt format` などから `internal/markdown` を再利用しやすい
  - ✅ ast-grep / linter で「ドメイン層から `os` / `io` を import 禁止」等の静的ルールを書きやすい（CLAUDE.md の方針に合致）
  - ❌ 立ち上がりに小さな雛形ファイルが多くなる
  - ❌ 早すぎる抽象化のリスク（パーサ層とドメイン層を分けすぎてシグネチャ調整コストが出る可能性）

### Option C: ハイブリッド（純ロジック 1 パッケージ + I/O / CLI / TUI を分離）

**いつ採用するか**: R7 の純粋性を最優先しつつ、Option B の細分化はやり過ぎだと感じる場合の妥協案。Brief の方針との親和性が最も高い。

- **構成例**:
  - `internal/kanban/`: 行パーサ + 境界判定 + 親子判定 + 再構築 を 1 つのパッケージに集約。エクスポート関数は `Sort(input string) (string, error)` のみ（内部は `parser.go` / `tree.go` / `move.go` 等のファイル分割）
  - `internal/diffview/`: dry-run 用の差分文字列生成（純関数）
  - `internal/fileio/`: 改行コード保持・アトミック書き込み（副作用層）
  - `cmd/mdt/`: Cobra ルート / `sort` サブコマンド / TUI 起動
  - `internal/tui/`: Bubble Tea スケルトン
- **トレードオフ**:
  - ✅ 純ロジックの公開 API が `kanban.Sort` 1 本に絞られ、再利用と TDD が両立する
  - ✅ I/O・差分・CLI・TUI が独立し、ast-grep で「`internal/kanban` から `os` / `fmt`(stdout) 禁止」等の静的ルールを記述可能
  - ✅ ファイル数は Option B より少なく、ドメインの内的分割は file 単位に閉じる
  - ❌ `internal/kanban` の内部設計（パーサとドメインを 1 パッケージに混ぜる）が後から「分けたい」となった場合、再分割の作業が発生
  - ❌ `diffview` を切り出すか `kanban` に同居させるかは設計時に再判断が必要

#### 推奨の方向性

- **第一候補は Option C**。理由は以下:
  - R7（純粋性 / 決定性 / I/O 分離）を最少のパッケージ数で満たせる。
  - brief の Boundary Candidates（5 層）との整合は「同一パッケージ内のファイル分割」で十分達成でき、Option B のオーバーヘッドを避けられる。
  - 将来サブコマンドが増えた段階で `internal/kanban` から `internal/markdown`（純パーサ）を切り出すリファクタは低コストで実施可能（公開 API は `kanban.Sort` のままに保てる）。
- Option B は「複数のマークダウン系サブコマンドを同時に着手する」計画が固まった段階で再評価する。

## 5. 外部依存とライセンス整合

| 項目 | 状況 | リスク |
| --- | --- | --- |
| Go 1.25+ | brief で確定 | Low（前提として固定） |
| `github.com/spf13/cobra` v1.10+ | brief で確定（未導入） | Low |
| `github.com/charmbracelet/bubbletea` v1 系 | brief で確定（未導入） | Low |
| ライセンス（Cobra: Apache-2.0、Bubble Tea: MIT） | brief で同梱可能と明示 | Low（NOTICE / LICENSE 同梱方針を design で確認） |
| diff 表示用ライブラリ | 未確定 | Medium（後述「Research Needed」） |

## 6. Research Needed（design フェーズへの持ち越し）

> 解析の段階で深掘りせず、design 段階で確定させたい項目を列挙する。

1. **`--dry-run` の差分フォーマット**:
   - 候補: 自前で `+` / `-` プレフィックスを付ける最小フォーマット / `github.com/sergi/go-diff` 等の外部ライブラリ / `diff` コマンド出力風の unified diff
   - 決定軸: 依存追加コスト・出力可読性・テスト容易性
2. **アトミック書き込み戦略**:
   - 候補: 同一ディレクトリに一時ファイルを作成し `os.Rename` で置換 / `os.WriteFile` 直書き
   - 決定軸: R4-5（中間破損の禁止）と Windows 互換性
3. **改行コード保持ロジック**:
   - 入力末尾改行の有無検出、CRLF / LF の混在ファイルの扱い、TUI 経由の起動でも同一挙動か
4. **Bubble Tea スケルトン仕様**:
   - 表示する初期テキスト・キーバインド（`q` / `Ctrl+C`）・終了時の終了コード（R6-4 = 0）の最小実装
   - Cobra の `RunE` から `tea.NewProgram(...).Run()` を呼ぶ標準パターンの確認
5. **インデント判定の細部**:
   - 「Tab 1 個 = 半角 2 個以上」を同一深さとみなすか、別軸で扱うか
   - インデント幅が混在した場合（同一親配下で Tab と空白が混じる）の正規化規則
6. **エラー区分と終了コード**:
   - R5-3 の「使用方法不正 / 実行時エラー」を別の非ゼロコードに分けるか、一律 1 にするか
7. **Linter / ast-grep ルール**:
   - 「`internal/kanban` から `os` / `io` / `fmt`(stdout) を import 禁止」「`Sort` は決定性のため `time` / `rand` 不使用」等の静的ルール候補
8. **Justfile の最小レシピ**:
   - `just test` / `just lint` / `just build` / `just run` の初期セット
9. **テスト戦略**:
   - 純関数のテーブル駆動 + ゴールデンファイル
   - I/O 層のテンポラリディレクトリ駆動 (`t.TempDir()`)
   - TUI のヘッドレステスト（Bubble Tea の `teatest` 利用可否）
10. **ステアリング不在の解消**:
   - design 着手前に `/kiro-steering` で `product.md` / `tech.md` / `structure.md` を整備するかどうか

## 7. Effort & Risk

- **Effort: M（3–7 日）**
  - 理由: グリーンフィールドだが範囲は単一 CLI ＋ 最小 TUI スケルトンに閉じている。純ロジックは数百行規模で TDD 駆動可能。Cobra / Bubble Tea の雛形と `Justfile`・linter 設定の初期構築コストが乗るため、S では収まりにくい。
- **Risk: Medium**
  - 中リスク要因: 改行コード保持・アトミック書き込み・差分フォーマットの設計判断が複数選択肢を持つこと、ステアリング不在のまま着手することによる規約のブレ。
  - 低リスク要因: 外部統合がなく、ロジックは決定的・純粋。テストで境界網羅が容易。

## 8. Recommendations for Design Phase

1. **構成は Option C（純ロジック 1 パッケージ + I/O / CLI / TUI 分離）を起点に design する**。Option B への昇格は将来サブコマンド追加時の再判断とする。
2. **公開 API を `kanban.Sort(input string) (string, error)` の 1 本に固定**し、design でシグネチャを最初に確定させる（残りの内部分割はリファクタ余地として確保）。
3. **「Research Needed」7 項目を design.md の前半で意思決定する**。特に diff フォーマット・アトミック書き込み・改行コード保持の 3 点は実装着手前に確定が必須。
4. **design 着手前に `/kiro-steering` を実行**して `tech.md`（Go 1.25 / Cobra / Bubble Tea / `just` / TDD）と `structure.md`（Option C のレイアウト）を固定すると、後続タスクの整合確認コストが下がる。
5. **静的検査ルール（linter / ast-grep）を design.md に明記**し、CLAUDE.md の「静的検査可能なルールはプロンプトでなく linter / ast-grep」を満たす形で運用する。
