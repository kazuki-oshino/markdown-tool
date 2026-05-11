package kanban

// findBoundary は完了集と未完了パートの境界インデックスを返す。
//
// 仕様 (requirements.md R2.1, R2.2, R2.3 / design.md "完了集／未完了パート境界判定"):
//   - kindDivider 行（trim 後に "---" と一致する行）が存在する場合、最後の
//     kindDivider 行のインデックスを返す。これが「完了集の末尾の直後 =
//     未完了パートの先頭」となる。
//   - kindDivider 行が存在しない場合、最初の kindUnchecked 行（"[ ]" を含む行）
//     のインデックスを返す。これは既存の divider 無しファイルの境界規則である。
//   - kindDivider / kindUnchecked 行が 1 つも存在しなければ len(lines) を返す。
//     呼び出し側は b == len(lines) を「未完了パートなし」と判定して Sort の
//     早期リターン (R2.2 / R2.3) に用いる。
//   - 先頭行が kindUnchecked または kindDivider のとき 0 を返す。完了集が空（R2.3 相当）。
//   - 空 slice 入力では 0 == len(lines) を返す（境界なし、no-op 入力）。
//
// 純関数: lines を読むだけで副作用を持たず、line.kind のみを参照する。
// indent / raw / 行内容には依存しない（kind の事前確定は parseLines の責務）。
func findBoundary(lines []line) int {
	lastDivider := -1
	for i, l := range lines {
		if l.kind == kindDivider {
			lastDivider = i
		}
	}
	if lastDivider >= 0 {
		return lastDivider
	}
	for i, l := range lines {
		if l.kind == kindUnchecked {
			return i
		}
	}
	return len(lines)
}

// findInsertionIndex は完了集 (lines[:boundary]) のうち、最後の kindChecked
// ブロック（root + 子孫サブツリー）の直後インデックスを返す。
//
// 役割:
//   - 移動対象の完了ブロックを「完了集の中の最後の [x] サブツリーの直後」に挿入するための
//     位置を決定する。これにより [x] と [ ] の間にある空行 / 見出し等の kindOther 行は
//     挿入位置より後ろ（remaining 側）に残り、移動ブロックが空行を跨いで [ ] 直前まで
//     押し込まれる挙動を抑止する。
//
// アルゴリズム:
//  1. boundary-1 から逆順に走査し、最後の kindChecked 行 i を見つける。
//  2. その行の indent を baseIndent とし、i+1 から boundary-1 まで indent > baseIndent が
//     連続する範囲を子孫サブツリーとして吸収する (subtreeEnd を進める)。
//  3. subtreeEnd + 1 を返す。
//
// フォールバック:
//   - 完了集に kindChecked 行が 1 つも無い (例: 見出し行のみで [x] が無いケース) ときは
//     boundary をそのまま返す。この場合の出力は変更前と完全に同一になる。
//
// 純関数: lines を読むのみで副作用を持たない。
func findInsertionIndex(lines []line, boundary int) int {
	last := -1
	for i := boundary - 1; i >= 0; i-- {
		if lines[i].kind == kindChecked {
			last = i
			break
		}
	}
	if last < 0 {
		return boundary
	}
	base := lines[last].indent
	end := last
	for end+1 < boundary && lines[end+1].indent > base {
		end++
	}
	return end + 1
}
