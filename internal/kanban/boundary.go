package kanban

// findBoundary は完了集と未完了パートの境界インデックスを返す。
//
// 仕様 (requirements.md R2.1, R2.2, R2.3 / design.md "完了集／未完了パート境界判定"):
//   - 最初の kindUnchecked 行（"[ ]" を含む行）のインデックスを返す。
//     これが「完了集の末尾の直後 = 未完了パートの先頭」となる (R2.1)。
//   - kindUnchecked 行が 1 つも存在しなければ len(lines) を返す。
//     呼び出し側は b == len(lines) を「未完了パートなし」と判定して
//     Sort の早期リターン (R2.2 / R2.3) に用いる。
//   - 先頭行が kindUnchecked のとき 0 を返す。完了集が空（R2.3 相当）。
//   - 空 slice 入力では 0 == len(lines) を返す（境界なし、no-op 入力）。
//
// 純関数: lines を読むだけで副作用を持たず、line.kind のみを参照する。
// indent / raw / 行内容には依存しない（kind の事前確定は parseLines の責務）。
func findBoundary(lines []line) int {
	for i, l := range lines {
		if l.kind == kindUnchecked {
			return i
		}
	}
	return len(lines)
}
