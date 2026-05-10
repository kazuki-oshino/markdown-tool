// cmd/mdt の sort サブコマンド定義。
//
// 役割 (tasks.md 6.2 / requirements.md R1.1, R1.3, R1.5, R4.1, R4.3, R4.6,
// R5.1, R5.3, R5.4):
//   - `mdt sort <file>` を Cobra サブコマンドとして提供し、位置引数 1 個必須
//     (cobra.ExactArgs(1))・--dry-run フラグを宣言する (R5.1)
//   - RunE 内で fileio.Read → kanban.Sort → 結果分岐 (no-op / dry-run / write) を実装する
//     (design.md "System Flows / mdt sort 主要フロー")
//   - I/O 失敗は fmt.Errorf("...: %w", err) で wrap し、Cobra 経由で stderr に出力 →
//     classifyExitCode で ExitRuntimeError = 1 にマップされる (R1.5)
//   - 引数省略・引数過多・未知フラグは usageError でラップして classifyExitCode で
//     ExitUsageError = 2 にマップする (R5.3, R5.4)
//
// 使用方法ヒントの出力先:
//   - Cobra 既定の cmd.Usage() は OutOrStderr() (= SetOut が指定されていればそちら) を
//     使うため、テストで stdout バッファに使用方法が混ざる挙動になる。
//   - 本実装は cmd.UsageString() で使用方法文字列を取得し、ErrOrStderr() に明示的に
//     書き出す方針を採る (tasks.md 6.2 の「stderr へ出力」契約)。
//   - cmd.UsageString() は内部で writer を一時的に差し替えて usage を文字列として取得するため、
//     SetUsageFunc を上書きすると無限再帰になる (Usage → UsageString → Usage の循環)。
//     SetUsageFunc は Cobra 既定実装のまま使い、出力先制御は呼び出し側 (printUsageToStderr)
//     で行う。
//
// SilenceUsage / SilenceErrors:
//   - SilenceUsage = true: RunE の runtime error 時に usage を二重表示しない (R1.5 シナリオで
//     使用方法ヒントは不要、failure reason のみで十分)
//   - SilenceErrors = false: Cobra 標準のエラーメッセージ出力 ("Error: ...") は維持し、
//     上位 (Execute / classifyExitCode) では終了コード分類のみ責任を持つ
package main

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kazuki-oshino/markdown-tool/internal/diffview"
	"github.com/kazuki-oshino/markdown-tool/internal/fileio"
	"github.com/kazuki-oshino/markdown-tool/internal/kanban"
)

// newSortCommand は `sort` サブコマンドを構築する。
//
// 振る舞い:
//   - Args:   cobra.ExactArgs(1) を usageError でラップ。失敗時に使用方法を stderr へ出力
//   - Flag:   未知フラグは SetFlagErrorFunc で usageError ラップ + 使用方法を stderr へ出力
//   - RunE:   ファイル読み込み → ソート → no-op / --dry-run / 通常書き込みに分岐
func newSortCommand() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "sort <file>",
		Short: "未完了パートの完了行を完了集末尾へ並べ替える",
		Long: "sort は対象 Markdown ファイルを読み込み、未完了パート内の" +
			"移動可能な [x] ブロック (子孫を含む) を完了集末尾へ移動した結果で上書きする。\n" +
			"--dry-run を付けるとファイルを変更せず、差分プレビューを stdout に出力する。",

		// Cobra の RunE 由来エラーで usage を二重表示しないよう抑制する。
		// 引数バリデーション / フラグエラー時の使用方法表示は Args / FlagErrorFunc 内で
		// 明示的に cmd.Usage() を呼ぶことで stderr 経路を取る。
		SilenceUsage: true,

		// 位置引数 1 個必須の契約 (R5.1)。
		// cobra.ExactArgs(1) を直接 Args に渡すと使用方法が表示されないため、
		// ラップして ① 使用方法を stderr に出す ② usageError で分類する を両立させる。
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				printUsageToStderr(cmd)
				return &usageError{err: err}
			}
			return nil
		},

		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			// (1) ファイル読み込み (R1.5)。改行コード保持の Meta も併せて取得する。
			content, meta, err := fileio.Read(path)
			if err != nil {
				// fileio.Read の wrap には対象ファイルパスと操作種別が含まれているが、
				// CLI 層の文脈 (sort コマンドであること) を補強して上位に伝搬する。
				return fmt.Errorf("ファイル読み込みに失敗しました: %w", err)
			}

			// (2) 純ロジックでソート。LF 統一前提・副作用なし (design.md "改行コード責務の分割")。
			sorted, err := kanban.Sort(content)
			if err != nil {
				// 現状 MVP では nil のみ想定だが、将来の致命的不整合に備えて経路を確保する。
				return fmt.Errorf("ソートに失敗しました: %w", err)
			}

			// (3) 入力 == 出力なら書き込み・差分出力ともに行わず exit 0 (R1.3 / design.md no-op 分岐)。
			//     --dry-run よりも先に判定する (sequence diagram の alt 節先頭分岐に合わせる)。
			if sorted == content {
				return nil
			}

			// (4) --dry-run: 差分を stdout に出力し、ファイルは変更しない (R4.1, R4.6)。
			if dryRun {
				diff := diffview.Render(content, sorted)
				fmt.Fprint(cmd.OutOrStdout(), diff)
				return nil
			}

			// (5) 通常書き込み: アトミック上書きで改行コード Meta を再適用する (R4.3, R4.4, R4.5)。
			if err := fileio.AtomicWrite(path, sorted, meta); err != nil {
				return fmt.Errorf("ファイル書き込みに失敗しました: %w", err)
			}
			return nil
		},
	}

	// --dry-run フラグ (R4.1)。短縮形は付けない (tasks.md / design.md にも仕様なし)。
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "差分プレビューを stdout に出力し、ファイルを変更しない")

	// 未知フラグも UsageError 分類とし、使用方法を stderr に出す (R5.3)。
	// 親 root の FlagErrorFunc (root.go) は使用方法を出さないため、ここで上書きする。
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		printUsageToStderr(c)
		return &usageError{err: err}
	})

	return cmd
}

// printUsageToStderr は cmd の使用方法文字列を ErrOrStderr() に書き出す。
//
// 設計理由 (tasks.md 6.2 の stderr 契約):
//   - cmd.Usage() / SetUsageFunc 経由で出力先を変えると、cmd.UsageString() の
//     内部実装 (writer を一時的に差し替えて Usage() を呼ぶ) と循環し、
//     Usage → UsageString → Usage の無限再帰になる。
//   - そこで「Usage 文字列の取得は UsageString に委ね、出力先制御だけ呼出側で行う」
//     方針を取ることで、Cobra 既定の UsageFunc を温存しつつ stderr 出力を実現する。
//
// 末尾改行は UsageString が含めているため、Fprint で素直にそのまま流す。
func printUsageToStderr(cmd *cobra.Command) {
	fmt.Fprint(cmd.ErrOrStderr(), cmd.UsageString())
}
