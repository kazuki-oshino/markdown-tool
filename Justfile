# mdt の開発タスクランナ。
#
# 静的解析は task 1.2 で golangci-lint (depguard) + ast-grep に拡張済み。
# - depguard: internal/kanban から os/io/io/fs/os/exec/net/http/time/math/rand/crypto/rand を import 禁止
# - ast-grep: internal/kanban 配下の fmt.Print*/fmt.Fprint* を構文レベルで禁止 (fmt.Sprintf は許可)
# 両方とも tools/ 配下の設定ファイルを参照する (CLAUDE.md「静的検査可能なルールは linter / ast-grep」方針)。

# デフォルト一覧表示
default:
    @just --list

# 全テストを実行
test:
    go test ./...

# 静的解析: go vet + golangci-lint(depguard) + ast-grep
# 失敗時は最初に非ゼロ終了したコマンドの段階でレシピが停止する (just のデフォルト挙動)
lint:
    go vet ./...
    golangci-lint run --config tools/golangci/.golangci.yml ./...
    ast-grep scan --rule tools/ast-grep/no-io-in-kanban.yml

# 開発用ツールのインストール手順を表示する。
# 要求バージョン:
#   - golangci-lint: v2.x (動作確認済 2.12.2 / v2 形式の .golangci.yml が必須)
#   - ast-grep:      v0.42 以上 (動作確認済 0.42.1)
# macOS では Homebrew、それ以外は公式手順を参照すること。
setup-tools:
    @echo "[mdt] 必要ツールのインストール手順"
    @echo ""
    @echo "macOS (Homebrew):"
    @echo "  brew install golangci-lint ast-grep"
    @echo ""
    @echo "Linux / その他 (公式手順):"
    @echo "  golangci-lint: https://golangci-lint.run/welcome/install/  (v2.x を選択)"
    @echo "  ast-grep:      https://ast-grep.github.io/guide/quick-start.html  (v0.42 以上)"
    @echo ""
    @echo "要求バージョン:"
    @echo "  - golangci-lint v2.x  (本リポジトリの設定は v2 形式)"
    @echo "  - ast-grep      v0.42 以上"

# mdt をビルドし、$GOPATH/bin にもインストールする（どこからでも `mdt` で呼べる）
# 前提: `$(go env GOPATH)/bin` が PATH に通っていること
# - ./mdt: 動作確認・smoke test 用にリポジトリ直下へ出力
# - $GOPATH/bin/mdt: PATH 経由で呼び出せる本体
build:
    go build -o mdt ./cmd/mdt
    go install ./cmd/mdt

# 開発実行（引数なし起動）
run:
    go run ./cmd/mdt

# Go ソースを整形
fmt:
    go fmt ./...
