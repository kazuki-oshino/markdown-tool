// Package kanban は対象ファイル外への I/O を一切持たない純ドメイン層。
// 対外公開 API（Sort 等）は後続タスク（2.5）で追加する。task 2.1 では
// 行パーサとインデント深さ計算 (parseLines / indentDepth) のみを置き、
// 行種別 enum (kindOther / kindUnchecked / kindChecked) と
// 値型 line を確定する。
//
// 設計上の不変条件:
//   - 入力は LF (\n) 統一文字列前提。CRLF / 末尾改行の保持は fileio 層の単独責務。
//   - 本パッケージは os / io / time / rand / fmt.Print* 等を使わない（depguard / ast-grep で機械検証）。
package kanban

import "strings"

// kind は 1 行が表す checkbox 種別。
type kind int

const (
	// kindOther はチェックボックスを含まない通常行（見出し / 空行 / プレーンテキスト等）。
	kindOther kind = iota
	// kindUnchecked は未完了タスク行（"[ ]" を含む）。
	kindUnchecked
	// kindChecked は完了タスク行（"[x]" または "[X]" を含む）。
	kindChecked
)

// line はパース済み 1 行の値オブジェクト。
//   - indent: indentDepth(raw) の結果（design.md "インデント混在の正規化規則" を満たす）。
//   - kind:   行の checkbox 種別。
//   - raw:    LF 分割後の素の文字列（改行文字を含まない）。後段の assemble で原文を復元するため保持する。
type line struct {
	indent int
	kind   kind
	raw    string
}

// parseLines は LF 統一済みの入力を 1 行ずつ line に分解する。
//
// 仕様:
//   - strings.Split(input, "\n") の結果と完全 1:1 対応する。末尾改行があれば最終要素として
//     空行 (raw = "") が付き、strings.Join(raws, "\n") で round-trip 可能。
//   - 各行の indent は indentDepth(raw) を呼び出して算出する。
//   - 各行の kind は次の優先順位で判定する: "[x]" or "[X]" を含む → kindChecked、
//     "[ ]" を含む → kindUnchecked、いずれも含まない → kindOther。
//   - チェックボックスの位置（行頭からの距離・リストマーカ "- " 等の有無）は問わない。
//     現実の kanban Markdown では 1 行に複数のチェックボックスは出現しない前提。
//
// 副作用は持たない（標準ライブラリ strings のみに依存）。
func parseLines(input string) []line {
	raws := strings.Split(input, "\n")
	out := make([]line, len(raws))
	for i, r := range raws {
		out[i] = line{
			indent: indentDepth(r),
			kind:   classifyKind(r),
			raw:    r,
		}
	}
	return out
}

// indentDepth は raw の行頭の連続空白を左から走査し、深さを整数で返す。
//
// 正規化規則 (design.md "インデント混在の正規化規則"):
//   - タブ文字 '\t' は 1 個ごとに深さ +1。
//   - 半角スペース ' ' は連続 2 個ごとに深さ +1。端数 1 個は深さに加算しない（半角 1 個は深さ 0 と同等扱い）。
//   - タブと半角の混在: タブ優先解釈。タブが現れた以降の半角は同一段の継続とみなし、深さに加算しない。
//   - 非空白文字を検出した時点で走査終了。
//
// 純関数: 同一 raw に対し常に同一の整数を返す。
func indentDepth(raw string) int {
	depth := 0
	pendingSpaces := 0
	postTab := false
	for _, r := range raw {
		switch r {
		case '\t':
			depth++
			pendingSpaces = 0
			postTab = true
		case ' ':
			if postTab {
				// タブ後に続く半角は同一段の継続として深さに加算しない。
				continue
			}
			pendingSpaces++
			if pendingSpaces == 2 {
				depth++
				pendingSpaces = 0
			}
		default:
			// 非空白文字: ここで走査終了。以降のタブ・スペースは深さに含めない。
			return depth
		}
	}
	return depth
}

// classifyKind は raw の checkbox 種別を判定する。
// design.md / tasks.md 2.1 の規則に従い「完了優先」で判定する
// （[x]/[X] と [ ] が同一行に共存するケースは現実の kanban Markdown では想定外）。
func classifyKind(raw string) kind {
	if strings.Contains(raw, "[x]") || strings.Contains(raw, "[X]") {
		return kindChecked
	}
	if strings.Contains(raw, "[ ]") {
		return kindUnchecked
	}
	return kindOther
}
