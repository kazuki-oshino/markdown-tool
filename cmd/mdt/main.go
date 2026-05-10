// Package main は mdt CLI のエントリポイント。
//
// 役割:
//   - newRootCommand で構築した Cobra ルートを Execute する
//   - 戻り値の終了コード分類に従って os.Exit する (R5.3)
//
// 注意: cmd/mdt は CLI Adapter 層であり、os import は許容される。
// internal/kanban のような純ロジック層からの os import 禁止ルール (R7.3) には触れない。
package main

import "os"

// main は Cobra ルートコマンドを構築・実行し、終了コードに応じて os.Exit する。
//
// Execute からの戻り値 error は Cobra が既に stderr へ出力済みのため、
// ここでは終了コードのみを伝搬する責務に限定する。
func main() {
	code, _ := Execute()
	os.Exit(code)
}
