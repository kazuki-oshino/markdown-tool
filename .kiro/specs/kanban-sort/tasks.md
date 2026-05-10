# Implementation Plan

## 1. プロジェクト基盤の整備

- [x] 1.1 Go モジュールと Justfile / .gitignore の初期化
  - `go mod init` で Go 1.25 モジュールを定義する（モジュールパスはリポジトリに合わせて確定）
  - `Justfile` に `just test` / `just lint` / `just build` / `just run` / `just fmt` の最小レシピを追加
  - `.gitignore` に `mdt` バイナリ・`coverage.out`・IDE 設定（`.vscode/`、`.idea/`）を追記
  - `just build` が空の `cmd/mdt/main.go`（最小スタブ）でコンパイル成功し、`mdt` バイナリを生成する
  - _Requirements: 1.4_

- [x] 1.2 静的解析ルール（depguard + ast-grep）の整備
  - `tools/golangci/.golangci.yml` に depguard を設定し、`internal/kanban` から `os` / `io` / `io/fs` / `os/exec` / `net/http` / `time` / `math/rand` / `crypto/rand` の import を禁止
  - `tools/ast-grep/no-io-in-kanban.yml` に `internal/kanban` 配下の `fmt.Print*` / `fmt.Fprint*` 検出ルールを記述（`fmt.Sprintf` は許可）
  - `Justfile` に `just lint` レシピを追加し、`golangci-lint`（depguard 経由）と `ast-grep` 双方を呼び出す
  - `Justfile` に `just setup-tools` レシピ（または README 相当のコメント）を追加し、`golangci-lint` / `ast-grep` のインストール方法と要求バージョンを明示する
  - 故意に違反 import を入れたダミーファイルで `just lint` が非ゼロ終了することを確認後、ダミーは削除して状態を元に戻す
  - _Requirements: 1.4, 7.2, 7.3, 7.4_

- [x] 1.3 ゴールデンテスト用 testdata フィクスチャの準備
  - `testdata/kanban/` に kanban 層用の LF 統一フィクスチャ（`basic_unchanged` / `single_x_move` / `parent_child_all_done` / `parent_done_child_open` / `child_done_parent_open`）の `*.md.in` / `*.md.out` ペアを作成
  - kanban 配下のフィクスチャはすべて LF 改行のみとし、CRLF や Mixed は混入させない（kanban.Sort の LF 統一前提と整合）
  - `testdata/fileio/` に fileio 層用の `crlf_preserve.md.in` / `.md.out` を配置し、CRLF 改行で実保存（design.md File Structure Plan の更新後レイアウトに整合）
  - `.gitattributes` に `testdata/fileio/*.md.in -text` / `testdata/fileio/*.md.out -text` を追加し、Git の autocrlf 変換から fileio フィクスチャを保護
  - `testdata/kanban` と `testdata/fileio` 双方で `*.md.in` と `*.md.out` の件数が一致することを確認
  - _Requirements: 7.2_

- [x] 1.4 外部依存パッケージの追加
  - `go get github.com/spf13/cobra@latest` で Cobra を追加し、解決された実バージョンを `go.mod` に固定する（v1.10 リリース未定のため `@latest` の実測値を採用）
  - `go get github.com/charmbracelet/bubbletea@latest` で Bubble Tea を追加し、解決された v1 系のバージョンを `go.mod` に固定
  - `go get github.com/charmbracelet/x/exp/teatest@latest` でテスト用ライブラリを追加
  - `go mod tidy` を実行して `go.mod` / `go.sum` を整列
  - `go build ./...` がスタブ状態で成功し、`go.mod` に 3 パッケージの具体バージョンが記録され、`go.sum` に対応エントリが含まれること
  - _Requirements: 1.4_

## 2. internal/kanban: 純ドメイン実装

- [ ] 2.1 行パーサと indentDepth 正規化
  - `internal/kanban/parser.go` に `line` / `kind`（`kindOther` / `kindUnchecked` / `kindChecked`）と `parseLines(input string) []line` / `indentDepth(raw string) int` を実装
  - インデント正規化規則: タブ 1 個 = 深さ +1、半角スペース 2 個ごとに +1、半角 1 個は深さ 0、タブと半角混在はタブ優先で同段継続
  - `[x]` / `[X]` を `kindChecked`、`[ ]` を `kindUnchecked`、それ以外を `kindOther` に分類
  - `parser_test.go` のテーブルテストで「タブ単独」「半角 2」「半角 4」「半角 1（無効）」「タブ+半角混在」の各ケースが期待 depth と kind を返すこと
  - _Requirements: 3.1_

- [ ] 2.2 完了集／未完了パート境界判定
  - `internal/kanban/boundary.go` に `findBoundary(lines []line) int` を実装
  - 最初の `kindUnchecked` 行のインデックスを返し、無ければ `len(lines)` を返す
  - `boundary_test.go` で「中間に `[ ]` あり」「先頭が `[ ]`」「`[ ]` なし」「空ファイル」の各ケースで期待 index を返すこと
  - _Requirements: 2.1, 2.2, 2.3_

- [ ] 2.3 親子フォレスト構築（buildBlocks）
  - `internal/kanban/tree.go` に `block` 型と `buildBlocks(lines []line, from int) []block` を実装
  - `from` 以降の未完了パートを最浅インデント行をルートとする順序付きフォレストとして構築し、`child.indent > parent.indent` 行を子孫に吸収する
  - 未完了パートの全行（`[x]` / `[ ]` / `kindOther` ルート含む）がフォレスト要素として網羅されること
  - `tree_test.go` で「フラットなルート列」「ネスト 2 段」「兄弟と従兄弟混在」「`kindOther` ルート混在」の各ケースで期待構造が得られること
  - _Requirements: 3.2_

- [ ] 2.4 ブロック移動可否判定（canMove）
  - `internal/kanban/move.go` に `canMove(b block) bool` を実装
  - ルート自身が `kindChecked` かつ子孫の全 `[x]` / `[ ]` 行が `kindChecked` のとき true、それ以外 false（R3.3 と R3.4 を構造的に一括吸収）
  - `move_test.go` で「親子全完了」「親完了/子未完」「子完了/親未完」「ルートが `kindOther`」「単独 `[x]` ルート」の各ケースが期待真偽値を返すこと
  - _Requirements: 1.2, 3.3, 3.4, 3.5_

- [ ] 2.5 Sort 公開 API + assemble + ゴールデンテスト
  - `internal/kanban/move.go` に `assemble(prefix []line, completed []block, remaining []line) string` を実装し、LF 統一文字列を組み立てる（改行コード復元は呼び出し側責務）
  - `internal/kanban/sort.go` に `Sort(input string) (string, error)` を実装し、`parseLines` → `findBoundary` → `buildBlocks` → 出現順走査（移動可ルートは `completed`、それ以外は `remaining`）→ `assemble` で結果を返す
  - 入力 == 出力（移動対象なし／未完了パートなし／完了集空）も等価文字列を返す（早期リターン）
  - 移動対象ブロックの内部行順序・インデント文字種・インデント幅を保持する
  - `sort_test.go` のゴールデンテストで `testdata/kanban/*.md.in` → `*.md.out`（LF 統一フィクスチャのみ）を全件 pass、かつ `Sort(Sort(x)) == Sort(x)` の冪等性が全ケースで成立
  - CRLF / Mixed フィクスチャは kanban 層では一切扱わず、`testdata/fileio/` 配下に分離されていること（design.md の責務分割と整合）
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.2, 2.3, 2.4, 2.5, 3.5, 3.6, 7.1, 7.2_

## 3. internal/diffview: 差分レンダラ

- [ ] 3.1 (P) LCS ベース unified-diff 風 Render
  - `internal/diffview/diff.go` に `Render(before, after string) string` を実装
  - 行ベース LCS で対応関係を求め、追加 `+` / 削除 `-` / 不変 ` ` プレフィックスを付与
  - 不変連続範囲はハンク前後 3 行のみを残し、`@@ -X,Y +X,Y @@` ヘッダ風セパレータで圧縮
  - `before == after` のとき空文字列を返す
  - `diff_test.go` のゴールデンテストで「移動 1 件」「移動複数件」「no-op（空文字列）」「親子ブロック移動」が期待出力と完全一致
  - _Requirements: 4.1, 4.2_
  - _Boundary: internal/diffview_
  - _Depends: 1.1_

## 4. internal/fileio: ファイル I/O 副作用層

- [ ] 4.1 (P) Read + LineEnding 検出 + Meta 抽出
  - `internal/fileio/meta.go` に `Meta` 構造体と `LineEnding` enum（`LineEndingLF` / `LineEndingCRLF` / `LineEndingMixed`）を定義
  - `internal/fileio/read.go` に `Read(path string) (content string, meta Meta, err error)` を実装
  - 改行コードを検出し、`content` は LF 統一に正規化、`Meta` に LineEnding と HasTrailingEOL を格納（混在は最頻採用）
  - 読み取り失敗時は `fmt.Errorf("...: %w", err)` で wrap して返す
  - `read_test.go` で `t.TempDir()` を用い「LF」「CRLF」「Mixed」「末尾改行あり」「末尾改行なし」「存在しないファイル」の各ケースで期待 Meta とエラーが得られること
  - _Requirements: 1.5, 4.4_
  - _Boundary: internal/fileio_
  - _Depends: 1.1_

- [ ] 4.2 AtomicWrite + Meta 再適用
  - `internal/fileio/write.go` に `AtomicWrite(path string, content string, meta Meta) error` を実装
  - LF 統一の `content` を `Meta.LineEnding` に応じた改行コードに復元し、`Meta.HasTrailingEOL` に従って末尾改行を付加／除去
  - 同一ディレクトリに一時ファイルを生成 → `os.Rename` でアトミック置換
  - 失敗時は一時ファイルを除去し、`path` の元内容を保持する
  - `write_test.go` で「LF/CRLF 保持」「末尾改行あり/なしの保持」「権限なしディレクトリで失敗時に元ファイル不変」の各ケースが pass
  - _Requirements: 4.3, 4.4, 4.5_

## 5. internal/tui: Bubble Tea スケルトン

- [ ] 5.1 (P) Run + 初期画面 + キー終了 + teatest
  - `internal/tui/skeleton.go` に Bubble Tea Model/Update/View と公開関数 `Run() error` を実装
  - 初期画面に固定テキスト（例: `mdt - press q or Ctrl+C to quit`）を表示
  - `q` キーまたは `Ctrl+C` 入力で `tea.Quit` を返し、`Run` は nil を返す
  - `skeleton_test.go` で `teatest` を用い「初期画面表示」「`q` 押下で終了」「`Ctrl+C` で終了」の各ケースが pass
  - _Requirements: 6.2, 6.3_
  - _Boundary: internal/tui_
  - _Depends: 1.4_

## 6. cmd/mdt: CLI 統合

- [ ] 6.1 ルートコマンド + 終了コード + TUI 起動分岐
  - `cmd/mdt/exitcode.go` に `ExitOK = 0` / `ExitRuntimeError = 1` / `ExitUsageError = 2` を定義
  - `cmd/mdt/root.go` で Cobra ルートコマンドを構築し、`--help` / `-h` で利用方法を表示して終了コード 0 を返す
  - サブコマンド未指定 & 引数なしの場合は RunE 内で `tui.Run()` を呼び出し、戻り値 nil で終了コード 0
  - `cmd/mdt/main.go` で `Execute` を呼び出し、エラー有無に応じて適切な終了コードでプロセスを終了
  - `cobra.Command.SetArgs([]string{"--help"})` ベースのテストで usage が stdout に表示され exit 0、`SetArgs([]string{})` で TUI 起動分岐が呼ばれることを `tui.Run` のフェイク等で確認
  - _Requirements: 5.2, 5.3, 6.1, 6.4_
  - _Depends: 1.4, 5.1_

- [ ] 6.2 sort サブコマンド RunE 配線
  - `cmd/mdt/sort.go` に `sort` サブコマンドを定義し、位置引数 1 個必須（`cobra.ExactArgs(1)`）と `--dry-run` フラグを宣言
  - RunE 内で `fileio.Read` → `kanban.Sort` を呼び、入力 == 出力なら no-op で exit 0 / `--dry-run` 指定なら `diffview.Render` を stdout 出力 / それ以外は `fileio.AtomicWrite` で上書き
  - 読み取り／書き込みエラーを stderr へ出力し `ExitRuntimeError = 1` で終了
  - 引数省略時・未知フラグ時は Cobra のバリデーションで usage を stderr へ出力し `ExitUsageError = 2`
  - `SetArgs` テストで `mdt sort path` / `mdt sort --dry-run path` / `mdt sort`（引数省略）/ `mdt sort 存在しないファイル` の終了コードと stdout/stderr が期待通り
  - _Requirements: 1.1, 1.3, 1.4, 1.5, 4.1, 4.3, 4.6, 5.1, 5.4_
  - _Depends: 1.4, 2.5, 3.1, 4.2_

## 7. End-to-End 検証

- [ ] 7.1 sort サブコマンドの統合テスト
  - `cmd/mdt/sort_integration_test.go` で `testdata/fileio/crlf_preserve.md.in` を `t.TempDir()` 配下にコピーし CRLF E2E ケースとして利用、LF / 末尾改行有無のサンプルは TempDir 配下にインライン生成
  - `mdt sort <file>` 実行後にファイルが期待通り上書きされ、改行コード（LF / CRLF）と末尾改行有無が保持されていること
  - `mdt sort --dry-run <file>` 実行後にファイルが不変で、stdout が `diffview.Render` の期待出力と一致すること
  - 入力 == 出力ケースでファイル不変かつ exit 0 を返すこと
  - 全ケースが pass し、`go test ./cmd/mdt/...` が成功
  - _Requirements: 1.4, 4.3, 4.4, 4.6_
  - _Depends: 6.2_

- [ ] 7.2 静的解析と全テストの最終確認
  - `just lint` を実行し、depguard / ast-grep / `go vet` がすべて pass
  - `internal/kanban` 配下に意図せぬ I/O import（`os` / `io` / `time` / `rand` / `fmt.Print*` 等）が混入していないことを確認
  - `just test` で全テスト（`./...`）が pass
  - 全コマンドが exit 0 で終了し、CI 想定でのレディ状態を満たす
  - _Requirements: 1.4, 7.3, 7.4_
  - _Depends: 1.2, 6.2_

## Implementation Notes

- task 1.2: depguard (golangci-lint v2) の `files:` glob は `**/` の挙動が `glob.Glob` 仕様で「中間 segment が 1 つ以上必要」となるため、`**/internal/kanban/**/*.go` 単独では `internal/kanban/` 直下ファイルが拾えない。`**/internal/kanban/*.go` と `**/internal/kanban/**/*.go` の 2 系統を併記する必要がある (`tools/golangci/.golangci.yml` 内コメント参照)。
- task 1.2: ast-grep の Go パーサで `fmt.Println($$$)` という pattern は `type_conversion_expression` として解釈されてしまい呼び出し式にマッチしない。`kind: call_expression` + `regex: "^fmt\\.(Print|Println|Printf|Fprint|Fprintln|Fprintf)\\("` の組合せで構文ノードを呼び出し式に固定し regex で関数名を絞る方式が必須。
- task 1.2: `internal/kanban/` 配下に `_*.go` で始まるダミーファイルを置くと Go の build が無視するため lint 検証も発火しない。検証用ダミーは `zz_*.go` 等の通常ファイル名で配置し、検証後に必ず削除する。
- task 1.4: スタブ状態 (cmd/mdt/main.go が空) のままでは `go mod tidy` が cobra / bubbletea / teatest を未使用と判定して go.mod から削除する。`tools/depspin/deps.go` に `//go:build deps_pin` タグ付きの blank import アンカーを置くことで、デフォルトビルドからは除外しつつ tidy の解析対象として保持し、3 パッケージを direct require に固定できる。task 5.1 (bubbletea/teatest) と task 6.1 (cobra) の実 import が入った時点で `tools/depspin/` ディレクトリは削除して構わない。
- task 1.4: `github.com/charmbracelet/x/exp/teatest` は安定タグを持たない実験的パッケージで `@latest` 解決値が `v0.0.0-<timestamp>-<commit>` 形式の Go pseudo-version になる（例: `v0.0.0-20260510005209-39224119bc89`）。これは Go semver の正規形式であり、`go.mod` への固定として有効。リリース版を期待する記述（design 等）は禁止し、解決値そのままを `go.mod` に保存する運用とする。
