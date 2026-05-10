// 終了コード規約 (R5.3) の定数定義。
//
// design.md "CLI Adapter / cmd/mdt (root + sort)" の Service Interface に従い、
// 成功 = 0 / 実行時エラー = 1 / 使用方法不正 = 2 を一意に固定する。
// main.go はこれらの定数を介して os.Exit を呼ぶ。
package main

const (
	// ExitOK は正常終了を示す終了コード (R5.3 成功時)。
	ExitOK = 0
	// ExitRuntimeError はファイル読み書き失敗・TUI 起動失敗など、
	// 実行時エラーを示す終了コード (R5.3 実行時エラー時)。
	ExitRuntimeError = 1
	// ExitUsageError は引数不足・未知フラグなど、
	// 使用方法の不正を示す終了コード (R5.3 使用方法不正時)。
	ExitUsageError = 2
)
