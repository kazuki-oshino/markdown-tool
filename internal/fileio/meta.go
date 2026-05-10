// Package fileio は対象 Markdown ファイルの読み書きと改行コード保持・
// アトミック書き込みを提供する副作用境界。
//
// 改行コード責務分担 (design.md "改行コード責務の分割"):
//   - Read:        disk bytes → 改行コード検出 → LF 統一 content + Meta
//   - kanban.Sort: LF 統一文字列のまま処理（改行コード非関知）
//   - AtomicWrite: LF 統一 content + Meta → 元の改行コードに復元して書き込み
//
// 本層は cmd/mdt の sort サブコマンドからのみ呼び出される (design.md
// "依存方向と静的ルール")。internal/kanban / internal/diffview は純レイヤとして
// 本層 (および os, io 等の I/O 系標準ライブラリ) を一切 import しない。
package fileio

// LineEnding は改行コード種別を表す enum。
//
// requirements.md R4.4 と design.md "Data Models / File I/O Meta" の契約に従い、
// 入力ファイルの改行コード種別を 3 値で表現する。
//
// 「混在」は task 4.1 / design.md table の規約に従い、Read では「最頻採用」とする
// （read.go の detectLineEnding 参照）。LineEndingMixed enum 値は表現としては
// 残すが、本実装の Read からは原則として返却されない（純粋に LF・CRLF どちらか
// 一方のみが優勢な状態に常に潰される）。これは 4.2 の AtomicWrite 側で「Mixed
// 時は最頻採用で書き戻せばよい」という不変条件と整合する。
type LineEnding int

const (
	// LineEndingLF は \n のみ、または LF が最頻 / 同数 / ファイルに改行なしのとき。
	LineEndingLF LineEnding = iota
	// LineEndingCRLF は \r\n のみ、または CRLF が最頻のとき。
	LineEndingCRLF
	// LineEndingMixed は呼び出し側が「混在」を明示的に表現したい場合の予約値。
	// 本パッケージの Read は最頻採用ルールに従い LineEndingMixed を返さない。
	LineEndingMixed
)

// Meta は対象ファイルの改行コード種別と末尾改行有無を保持する。
//
// 仕様 (design.md "Data Models / File I/O Meta" / requirements.md R4.4):
//   - LineEnding:     改行コード種別。Read で検出され AtomicWrite で再適用される。
//   - HasTrailingEOL: 元ファイルが末尾改行を持つかどうか。AtomicWrite で復元される。
type Meta struct {
	LineEnding     LineEnding
	HasTrailingEOL bool
}
