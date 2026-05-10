package fileio

import (
	"fmt"
	"os"
	"strings"
)

// Read は path のファイルを読み込み、改行コード種別と末尾改行有無を Meta に格納し、
// LF 統一に正規化した content と Meta を返す。
//
// 仕様 (tasks.md 4.1 / design.md "Side Effects / internal/fileio" の Service Interface /
//
//	requirements.md R1.5, R4.4):
//
//   - 改行コードを検出: 純 LF / 純 CRLF / 混在の 3 系統を区別し、混在は最頻採用ルール
//     （CRLF > LF → CRLF, LF >= CRLF → LF）に従い 1 つに集約する。
//   - content は LF 統一: \r\n をすべて \n に置換した文字列を返す。
//   - HasTrailingEOL: 元バイト列が \n または \r\n で終端しているかを反映する。
//   - 読み取り失敗時は fmt.Errorf("...: %w", err) で wrap して返す（R1.5 の
//     失敗理由メッセージ要件と、上位 cmd/mdt 層での errors.Is(err, os.ErrNotExist)
//     などのハンドリングのため）。
func Read(path string) (string, Meta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		// R1.5: 失敗理由を含むエラーを上位レイヤへ伝搬する。
		// %w を用いた wrap によって errors.Is/As でのカテゴリ判定 (e.g. os.ErrNotExist)
		// を可能にしておく。
		return "", Meta{}, fmt.Errorf("fileio: read %q: %w", path, err)
	}

	lineEnding, hasTrailingEOL := detectLineEnding(raw)
	// LF 統一: \r\n を \n に置換するだけで kanban / diffview 層へ渡せる形になる。
	// 単独の \r (CR) は通常の Markdown 運用ではほぼ発生しないため、本実装では
	// 行内バイトとしてそのまま保持する（行末分離としては扱わない）。
	content := strings.ReplaceAll(string(raw), "\r\n", "\n")

	return content, Meta{
		LineEnding:     lineEnding,
		HasTrailingEOL: hasTrailingEOL,
	}, nil
}

// detectLineEnding はバイト列を 1 パスで走査し、CRLF / LF の出現数と
// 末尾改行有無を判定して、最頻採用ルール込みで LineEnding を決定する。
//
// 戻り値:
//   - LineEnding:     CRLF が出現数で勝るとき LineEndingCRLF、それ以外 (LF 優勢 /
//     同数 / どちらも 0) は LineEndingLF。
//   - hasTrailingEOL: 末尾が \n (=> LF or CRLF どちらでも) かどうか。
//
// 実装メモ:
//   - 1 文字単位で走査し、\r\n を見つけたら CRLF を 1 加算してインデックスを 2 進める。
//   - 単独の \n は LF を 1 加算する。\r 単独 (LF が後続しない) は無視（行末分離としては数えない）。
//   - これにより CRLF と LF をダブルカウントせずに正確に区別できる。
func detectLineEnding(raw []byte) (LineEnding, bool) {
	var crlfCount, lfCount int
	for i := 0; i < len(raw); i++ {
		switch raw[i] {
		case '\r':
			if i+1 < len(raw) && raw[i+1] == '\n' {
				crlfCount++
				i++ // \n をスキップ（CRLF をひとまとまりとして数える）
			}
			// 単独 \r は本実装では行末として数えない（macOS Classic の旧仕様であり、
			// 現代の Markdown 運用では事実上発生しない）。
		case '\n':
			lfCount++
		}
	}

	hasTrailingEOL := len(raw) > 0 && raw[len(raw)-1] == '\n'

	switch {
	case crlfCount == 0 && lfCount == 0:
		// 改行を含まない / 空ファイル → LF をデフォルトとする (決定性のため)。
		return LineEndingLF, hasTrailingEOL
	case crlfCount > lfCount:
		// 純 CRLF を含むケースもこの分岐に流れる (lfCount == 0 のため)。
		return LineEndingCRLF, hasTrailingEOL
	default:
		// 純 LF / LF 優勢 / 同数: 同数時のタイブレークは LF とする
		// (read_test.go TestRead_LineEnding_Mixed_Tie の契約)。
		return LineEndingLF, hasTrailingEOL
	}
}
