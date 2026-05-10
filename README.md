# mdt (markdown-tool)

カンバン風 Markdown ファイル（上部=完了タスク集／下部=未完了 todo）を整理する CLI ツール。

## Getting Started

### 必要環境

- Go 1.25 以上
- [just](https://github.com/casey/just)（タスクランナ）

### セットアップ

```sh
git clone https://github.com/kazuki-oshino/markdown-tool.git
cd markdown-tool
just build
```

リポジトリ直下に `mdt` バイナリが生成されます。

### 使い方

#### `mdt sort <file>` — 完了行を並べ替える

未完了パート内の `[x]` ブロック（子孫含む）を完了集の末尾へ移動し、ファイルを上書きします。

```sh
./mdt sort path/to/kanban.md
```

差分だけ確認したい場合は `--dry-run` を付けてください（ファイルは変更されません）。

```sh
./mdt sort --dry-run path/to/kanban.md
```

#### `mdt`（引数なし）— TUI を起動

```sh
./mdt
```

### 開発タスク

```sh
just            # タスク一覧
just test       # 全テスト実行
just lint       # go vet + golangci-lint + ast-grep
just run        # 引数なしで起動（TUI）
just fmt        # go fmt
```

`just lint` には `golangci-lint` v2.x と `ast-grep` v0.42 以上が必要です。インストール手順は次で確認できます。

```sh
just setup-tools
```

### 終了コード

| Code | 意味 |
| --- | --- |
| 0 | 正常終了 |
| 1 | 実行時エラー（I/O 失敗など） |
| 2 | 使い方の誤り（未知のフラグ・引数不足など） |
