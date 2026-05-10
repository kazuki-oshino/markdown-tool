//go:build deps_pin

// Package depspin は、task 1.4 で外部依存を go.mod / go.sum に固定するための
// アンカーパッケージ。
//
// `//go:build deps_pin` ビルドタグにより通常の `go build ./...` 経路からは除外される
// ため、ランタイムバイナリには一切影響しない。一方で `go mod tidy` は build tag を
// またいで import グラフを走査するため、ここでの blank import が
// `github.com/spf13/cobra` / `github.com/charmbracelet/bubbletea` /
// `github.com/charmbracelet/x/exp/teatest` を `go.mod` の require に保持する手掛かりとなる。
//
// 後続タスク（task 5.1 で `internal/tui/skeleton.go` が bubbletea を、
// task 5.1 のテストが teatest を、task 6.1 で `cmd/mdt/root.go` が cobra を実 import した時点）で、
// 本ファイルは依存固定アンカーとしての役割を終える。
// その時点で `tools/depspin/` ディレクトリごと削除して構わない（task 6.x 完了時のクリーンアップ候補）。
package depspin

import (
	// 本 spec の Allowed Dependencies に対応するアンカー（実バージョンは go.mod 側で `@latest` 解決値に固定）。
	_ "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/x/exp/teatest"
	_ "github.com/spf13/cobra"
)
