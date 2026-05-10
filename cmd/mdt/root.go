// cmd/mdt のルートコマンド定義と TUI 起動分岐、終了コード分類ロジック。
//
// 役割:
//   - Cobra ルートコマンド `mdt` を構築する (R5.1, R5.2)
//   - サブコマンド未指定 & 引数なし時に TUI スケルトンを起動する (R6.1, R6.4)
//   - フラグ解析エラー（未知フラグ等）を usageError でラップし、
//     main 側で ExitUsageError = 2 にマップ可能にする (R5.3)
//
// テスタビリティ:
//   - tui.Run の直接呼び出しは TTY を要求するため、パッケージ変数 runTUI を介して
//     差し替え可能にしている。これによりテストでは fake を注入できる
//     (root_test.go と design.md "Implementation Notes / Risks" を参照)。
package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kazuki-oshino/markdown-tool/internal/tui"
)

// runTUI は TUI スケルトンを起動するインジェクション可能な関数変数。
//
// 通常運用では tui.Run をそのまま呼ぶが、テストでは fake に差し替えることで
// TTY 不在環境でも RunE の TUI 起動分岐を検証できる (R6.1, R6.4 の検証手段)。
var runTUI = tui.Run

// usageError は Cobra のフラグ解析エラー等、利用者の使い方の不正に起因するエラーを
// 通常の実行時エラーから区別するためのラッパ型。
//
// classifyExitCode はこの型を ExitUsageError = 2 にマップする (R5.3)。
type usageError struct {
	err error
}

// Error は内包するエラーのメッセージをそのまま返す。
func (e *usageError) Error() string {
	if e == nil || e.err == nil {
		return "usage error"
	}
	return e.err.Error()
}

// Unwrap は errors.Is / errors.As 経由で内包エラーを取り出せるようにする。
func (e *usageError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.err
}

// isUsageError は与えられたエラーが usageError でラップされているかを判定する。
func isUsageError(err error) bool {
	var ue *usageError
	return errors.As(err, &ue)
}

// classifyExitCode は Cobra Execute の戻り値を終了コードへ写像する (R5.3)。
//
//   - nil          → ExitOK (0)
//   - usageError   → ExitUsageError (2)
//   - それ以外      → ExitRuntimeError (1)
func classifyExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	if isUsageError(err) {
		return ExitUsageError
	}
	return ExitRuntimeError
}

// newRootCommand は `mdt` ルートコマンドを構築する。
//
// 振る舞い:
//   - --help / -h: Cobra 標準挙動で usage を Out (= stdout) に出力し、エラー nil で帰る (R5.2)
//   - サブコマンド未指定 & 引数なし: RunE 内で runTUI を呼ぶ (R6.1, R6.4)
//   - 未知フラグ: SetFlagErrorFunc で usageError にラップして返す (R5.3)
//
// SilenceUsage / SilenceErrors:
//   - SilenceUsage = true:  RunE 由来の実行時エラー時に usage を二重出力しないよう抑制
//   - SilenceErrors = false: Cobra 標準のエラーメッセージ出力は維持し、main 側では
//     終了コード分類のみ責任を持つ
func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mdt",
		Short: "mdt はカンバン風 Markdown を整理するツール",
		Long: "mdt はカンバン風 Markdown ファイル（上部=完了タスク集／下部=未完了 todo）を" +
			"整理する CLI ツール。引数なしで起動すると TUI スケルトンが立ち上がる。",
		// 引数 0 個を許可（引数なし起動経路を保つ）。
		// サブコマンド（将来の `mdt sort` 等）は引数を自分で検証する。
		Args: cobra.NoArgs,
		// SilenceUsage を true にすることで、RunE で実行時エラーを返した際の
		// usage 二重表示を抑止する (Cobra 標準の挙動: RunE エラー時に usage を出す)。
		// --help はこの設定の影響を受けず正常に出力される。
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// サブコマンド未指定 & 引数なし → TUI 起動 (R6.1)
			// args は cobra.NoArgs により空保証されているが、防御的に len(args) == 0 を確認する。
			if len(args) == 0 {
				if err := runTUI(); err != nil {
					// TUI 起動失敗は実行時エラー (R6.4 補足: nil 戻り → exit 0)。
					// usageError でラップしないため classifyExitCode で ExitRuntimeError = 1 になる。
					return fmt.Errorf("tui の起動に失敗しました: %w", err)
				}
				return nil
			}
			// ここには到達しないが、将来のための保険としてヘルプ相当の動作にする。
			return cmd.Help()
		},
	}

	// フラグ解析エラー（未知フラグ等）を usageError でラップする (R5.3)。
	// Cobra 標準では FlagErrorFunc は元のエラーをそのまま返すため、
	// ここで分類タグ付けを行うことで main 側の終了コード判定を一意にする。
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return &usageError{err: err}
	})

	return cmd
}

// Execute はルートコマンドを構築して実行し、終了コードと終了時エラーを返す。
//
// 戻り値:
//   - int:   ExitOK / ExitRuntimeError / ExitUsageError のいずれか (R5.3)
//   - error: 元の error（main 側で stderr に出す責務はここでは持たない。
//            Cobra が SilenceErrors=false により既に stderr へ出力済み）
//
// main.go はこのコードで os.Exit を呼ぶ。
func Execute() (int, error) {
	cmd := newRootCommand()
	err := cmd.Execute()
	return classifyExitCode(err), err
}
