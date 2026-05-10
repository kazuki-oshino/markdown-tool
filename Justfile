# mdt の開発タスクランナ。
# task 1.1 では最小レシピのみを提供する。後続タスクで lint を golangci-lint + ast-grep に拡張する。

# デフォルト一覧表示
default:
    @just --list

# 全テストを実行
test:
    go test ./...

# 静的解析（task 1.2 で golangci-lint + ast-grep に置き換え予定の最小スタブ）
lint:
    go vet ./...

# mdt バイナリをリポジトリ直下にビルド
build:
    go build -o mdt ./cmd/mdt

# 開発実行（引数なし起動）
run:
    go run ./cmd/mdt

# Go ソースを整形
fmt:
    go fmt ./...
