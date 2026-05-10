package kanban

import "testing"

// TestFindBoundary は完了集／未完了パートの境界判定 (findBoundary) を検証する。
//
// 仕様 (tasks.md 2.2 / requirements.md R2.1, R2.2, R2.3, design.md):
//   - findBoundary は最初の kindUnchecked 行のインデックスを返す。
//   - kindUnchecked が 1 つも存在しなければ len(lines) を返す（未完了パート無し = 完了集のみ）。
//   - 先頭が kindUnchecked なら 0 を返す（完了集が空）。
//   - 空 slice は len(lines) == 0 を返す（境界なし、no-op 入力扱い）。
//
// 設計上の意図:
//   - 純関数。lines を読むのみで状態を持たない。
//   - kind の判定は parseLines 段階で確定済みのため、本関数は line.kind のみを参照する。
func TestFindBoundary(t *testing.T) {
	t.Parallel()

	// テストケースを構築するヘルパ。indent / raw は findBoundary が参照しないため
	// 任意値で良いが、可読性のため raw に元行のイメージを残す。
	mk := func(k kind, raw string) line {
		return line{indent: 0, kind: k, raw: raw}
	}

	cases := []struct {
		name  string
		lines []line
		want  int
	}{
		{
			// ケース 1: 中間に [ ] あり
			//   [x] / [x] / "見出し等" / [ ] / [x]
			//   → 完了集は index 0..2、未完了パート先頭は index 3。
			name: "unchecked_in_middle",
			lines: []line{
				mk(kindChecked, "- [x] done A"),
				mk(kindChecked, "- [x] done B"),
				mk(kindOther, "## 区切り"),
				mk(kindUnchecked, "- [ ] todo C"),
				mk(kindChecked, "- [x] done in unchecked area"),
			},
			want: 3,
		},
		{
			// ケース 2: 先頭が [ ]
			//   完了集が空、ファイル冒頭から未完了パート開始。
			name: "unchecked_at_head",
			lines: []line{
				mk(kindUnchecked, "- [ ] first"),
				mk(kindChecked, "- [x] later"),
			},
			want: 0,
		},
		{
			// ケース 3: [ ] なし
			//   未完了パートが存在しない → len(lines) を返す。
			//   呼び出し側は Sort 早期リターン (R2.2) を判定可能。
			name: "no_unchecked",
			lines: []line{
				mk(kindChecked, "- [x] done A"),
				mk(kindOther, "メモ"),
				mk(kindChecked, "- [x] done B"),
			},
			want: 3,
		},
		{
			// ケース 4: 空ファイル
			//   parseLines("") は長さ 1 の空行を返すが、findBoundary は []line{} 直接入力でも
			//   破綻しないことを契約として確認する（呼び出し側の合成耐性）。
			name:  "empty_lines",
			lines: []line{},
			want:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := findBoundary(tc.lines)
			if got != tc.want {
				t.Errorf("findBoundary(%q) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}
