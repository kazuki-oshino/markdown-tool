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

// TestFindInsertionIndex は完了集内の挿入位置決定 (findInsertionIndex) を検証する。
//
// 仕様:
//   - 完了集 (lines[:boundary]) の最後の kindChecked 行を見つけ、その行の indent より
//     深い後続の連続行（子孫サブツリー）を吸収した上で「サブツリー直後」のインデックスを返す。
//   - 完了集に kindChecked が 1 つも無ければ boundary をそのまま返す（フォールバック）。
//
// 設計上の意図:
//   - 「[x] と [ ] の間にある空行 / 見出し等の kindOther 行は、移動ブロックより後ろに残す」
//     という Sort の挿入位置契約を、Sort のゴールデンテストとは独立に固定する。
func TestFindInsertionIndex(t *testing.T) {
	t.Parallel()

	mk := func(indent int, k kind, raw string) line {
		return line{indent: indent, kind: k, raw: raw}
	}

	cases := []struct {
		name     string
		lines    []line
		boundary int
		want     int
	}{
		{
			// 完了集末尾に空行が複数あるケース。
			// 最後の [x] は index 1。空行 (index 2,3) は indent 0 で子孫扱いされない。
			// → 挿入位置は 2 (= [x] done B の直後、空行より前)。
			name: "trailing_blanks_after_last_checked",
			lines: []line{
				mk(0, kindChecked, "- [x] done A"),
				mk(0, kindChecked, "- [x] done B"),
				mk(0, kindOther, ""),
				mk(0, kindOther, ""),
				mk(0, kindUnchecked, "- [ ] open"),
			},
			boundary: 4,
			want:     2,
		},
		{
			// 最後の [x] にインデント深い子孫 (kindOther / kindChecked) が続くケース。
			// 子孫範囲は indent > 親 で連続するため吸収され、サブツリー直後を返す。
			name: "checked_with_indented_descendants",
			lines: []line{
				mk(0, kindChecked, "- [x] parent"),
				mk(1, kindOther, "  - sub note"),
				mk(1, kindChecked, "  - [x] child done"),
				mk(0, kindUnchecked, "- [ ] open"),
			},
			boundary: 3,
			want:     3,
		},
		{
			// 完了集に kindChecked が無い (見出し行のみ) ケース。
			// フォールバックで boundary をそのまま返す。
			name: "no_checked_in_prefix",
			lines: []line{
				mk(0, kindOther, "## Heading"),
				mk(0, kindUnchecked, "- [ ] open"),
			},
			boundary: 1,
			want:     1,
		},
		{
			// 最後の [x] が境界直前にあり、間に kindOther が無いケース。
			// 子孫もないため挿入位置は boundary と一致する。
			name: "checked_immediately_before_boundary",
			lines: []line{
				mk(0, kindChecked, "- [x] done"),
				mk(0, kindUnchecked, "- [ ] open"),
			},
			boundary: 1,
			want:     1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := findInsertionIndex(tc.lines, tc.boundary)
			if got != tc.want {
				t.Errorf("findInsertionIndex(%q) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}
