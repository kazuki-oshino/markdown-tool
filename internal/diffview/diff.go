// Package diffview は --dry-run 用の行ベース LCS unified-diff 風レンダラを提供する。
//
// 本パッケージは Pure View 層であり、副作用を持たない決定的な純関数のみを公開する。
// 入出力ともに LF 統一文字列を前提とし、CRLF 復元等は呼び出し側 (fileio 層) の責務。
//
// design.md "Pure View / internal/diffview" を契約レベルの参照点とする。
package diffview

import (
	"fmt"
	"strings"
)

// Render は before と after の行ベース LCS による差分を unified-diff 風
// (行単位 +/-/space プレフィックス、コンテキスト 3 行、@@ ハンクヘッダ) で返す。
// before == after の場合は空文字列を返す。
//
// 並べ替え行は LCS の都合で「削除位置に -, 挿入位置に +」として両方に出現するため、
// 移動の元位置と移動先が視覚的に識別できる (R4.2 の構造的保証)。
func Render(before, after string) string {
	if before == after {
		return ""
	}
	a := splitLines(before)
	b := splitLines(after)
	edits := lcsDiff(a, b)
	return formatHunks(edits, defaultContextLines)
}

// defaultContextLines はハンク前後に残す不変行数。
// design.md "Pure View" の出力規約 (3 行) に従う。
const defaultContextLines = 3

// splitLines は LF (\n) で分割し、末尾の空要素 (=末尾改行に由来) を除去する。
// 返り値の各要素は改行を含まない 1 行分の内容。
//
// 入力規約: before / after は LF 統一文字列であること (CRLF は呼出側違反)。
// 本関数は CR を区別せず文字として扱うため、CRLF が混入していた場合は
// 行末の \r が表示に乗る挙動となる。
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// editKind は edit script の 1 要素の種別。
type editKind int

const (
	editKeep   editKind = iota // 不変行 ( ` ` プレフィックス)
	editDelete                 // 削除行 (`-` プレフィックス)
	editInsert                 // 追加行 (`+` プレフィックス)
)

// edit は LCS 由来の 1 行差分。
type edit struct {
	kind editKind
	line string
}

// lcsDiff は a, b の LCS DP テーブルを構築し、バックトラックで edit script を生成する。
//
// tie-break は「up (delete) 優先」とする:
//
//	if dp[i-1][j] >= dp[i][j-1] -> go up (delete a[i-1])
//	else                          -> go left (insert b[j-1])
//
// この方針により「並べ替え行は元位置に -, 移動先に +」として現れ、
// design.md "Pure View" の出力例と整合する。
func lcsDiff(a, b []string) []edit {
	m, n := len(a), len(b)

	// dp[i][j] = a[:i] と b[:j] の LCS 長
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] >= dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// バックトラックで逆順に edit を積み、最後に反転する。
	edits := make([]edit, 0, m+n)
	i, j := m, n
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && a[i-1] == b[j-1]:
			edits = append(edits, edit{kind: editKeep, line: a[i-1]})
			i--
			j--
		case i > 0 && (j == 0 || dp[i-1][j] >= dp[i][j-1]):
			edits = append(edits, edit{kind: editDelete, line: a[i-1]})
			i--
		default:
			edits = append(edits, edit{kind: editInsert, line: b[j-1]})
			j--
		}
	}
	for l, r := 0, len(edits)-1; l < r; l, r = l+1, r-1 {
		edits[l], edits[r] = edits[r], edits[l]
	}
	return edits
}

// hunkItem は edit に元ファイル / 新ファイル上の行番号を付加した内部型。
//
//   - editKeep   : oldLine = 元ファイル上の 1 始まり位置, newLine = 新ファイル上の位置
//   - editDelete : oldLine = 元ファイル上の位置,           newLine = 0 (新ファイルには存在しない)
//   - editInsert : oldLine = 0 (元ファイルには存在しない), newLine = 新ファイル上の位置
type hunkItem struct {
	e       edit
	oldLine int
	newLine int
}

// formatHunks は edit script からコンテキスト ctx 行を残した unified-diff 風の
// ハンク列を生成する。変更が無い場合 (全 keep) は空文字列を返す。
//
// ハンク抽出ルール:
//
//   - 各変更 (delete / insert) を中心に前後 ctx 行の keep を含める
//   - 連続する変更領域の keep gap が 2*ctx 以下なら同一ハンクへ吸収する
//   - gap が 2*ctx を超えたら ctx 行ぶん切り出してハンクを閉じ、次のハンクへ進む
//   - 末尾 keep は ctx 行までしか残さない
//   - 連続するハンクは startPrev で重複しないように clamp する
func formatHunks(edits []edit, ctx int) string {
	if len(edits) == 0 {
		return ""
	}

	items := make([]hunkItem, len(edits))
	oldL, newL := 0, 0
	hasChange := false
	for i, e := range edits {
		var ol, nl int
		switch e.kind {
		case editKeep:
			oldL++
			newL++
			ol = oldL
			nl = newL
		case editDelete:
			oldL++
			ol = oldL
		case editInsert:
			newL++
			nl = newL
		}
		items[i] = hunkItem{e: e, oldLine: ol, newLine: nl}
		if e.kind != editKeep {
			hasChange = true
		}
	}
	if !hasChange {
		return ""
	}

	n := len(items)
	var sb strings.Builder
	i := 0
	endPrev := 0
	for i < n {
		// 先頭の keep を読み飛ばす (ハンク開始は最初の変更行から)
		for i < n && items[i].e.kind == editKeep {
			i++
		}
		if i >= n {
			break
		}

		// ハンク開始位置: 変更行の ctx 行手前。直前ハンク末尾を超えないよう clamp。
		start := max(i-ctx, endPrev, 0)

		// ハンク末尾を求めるため変更を貪欲に拡張する
		end := i
		for end < n {
			if items[end].e.kind != editKeep {
				end++
				continue
			}
			// 連続する keep の長さを計る
			j := end
			for j < n && items[j].e.kind == editKeep {
				j++
			}
			keepRun := j - end
			if j == n {
				// edit script 末尾の keep run。ctx 行まで残して打ち切り。
				if keepRun > ctx {
					end += ctx
				} else {
					end = j
				}
				break
			}
			// 中間 keep run。2*ctx を超えるならハンクを分割する。
			if keepRun > 2*ctx {
				end += ctx
				break
			}
			end = j
		}
		if end > n {
			end = n
		}

		emitHunk(&sb, items[start:end])
		endPrev = end
		i = end
	}
	return sb.String()
}

// emitHunk は hunk の `@@ -oldStart,oldCount +newStart,newCount @@` ヘッダと
// 各行 (` ` / `-` / `+` プレフィックス + 行内容 + LF) を sb に書き出す。
func emitHunk(sb *strings.Builder, h []hunkItem) {
	if len(h) == 0 {
		return
	}

	var oldStart, oldCount, newStart, newCount int
	for _, it := range h {
		if it.e.kind != editInsert {
			if oldCount == 0 {
				oldStart = it.oldLine
			}
			oldCount++
		}
		if it.e.kind != editDelete {
			if newCount == 0 {
				newStart = it.newLine
			}
			newCount++
		}
	}
	// 全 insert / 全 delete のハンクは本実装の抽出経路 (常に ctx 行のコンテキストを含めようとする)
	// では通常発生しない。ただし片側が空ファイルのときは発生し得るため、unified-diff 慣習に従い
	// 「直前行 = 0」を採用する。
	if oldCount == 0 {
		oldStart = 0
	}
	if newCount == 0 {
		newStart = 0
	}

	fmt.Fprintf(sb, "@@ -%d,%d +%d,%d @@\n", oldStart, oldCount, newStart, newCount)
	for _, it := range h {
		switch it.e.kind {
		case editKeep:
			sb.WriteByte(' ')
		case editDelete:
			sb.WriteByte('-')
		case editInsert:
			sb.WriteByte('+')
		}
		sb.WriteString(it.e.line)
		sb.WriteByte('\n')
	}
}
