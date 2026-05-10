// task 2.3: tree.go の buildBlocks に対するテーブルテスト。
// design.md "走査と移動アルゴリズム" / "Domain Model" を契約として固定する。
package kanban

import (
	"reflect"
	"testing"
)

// TestBuildBlocks は buildBlocks(lines, from) が lines[from:] を順序付きフォレストとして
// 構築する不変条件を網羅する (tasks.md 2.3 / requirements.md R3.2 / design.md "走査と移動アルゴリズム")。
//
// 不変条件:
//   - lines[from:] の全行（kindOther / kindUnchecked / kindChecked）が
//     フォレストの要素 (root or descendant) として網羅される。
//   - child.indent > parent.indent の連続を当該 root の子孫範囲として吸収する。
//   - 子孫範囲内では再帰的に同じルールでサブフォレストを構築する。
//   - 元の行順を保持する（root 順 / 各 children 順）。
//   - from が範囲外（負値・len(lines) 以上）のとき nil を返す。
//
// task 2.3 で必須の 4 系統「フラットなルート列」「ネスト 2 段」「兄弟と従兄弟混在」
// 「kindOther ルート混在」を網羅し、追加で from の境界・kindOther を子孫として吸収する
// ケースも決定論で固定する。
func TestBuildBlocks(t *testing.T) {
	t.Parallel()

	// テスト用 line 構築ヘルパ。indent / kind / raw を直接指定し parseLines をバイパスする
	// （buildBlocks の責務は parser とは独立に検証する）。
	mk := func(indent int, k kind, raw string) line {
		return line{indent: indent, kind: k, raw: raw}
	}

	cases := []struct {
		name  string
		lines []line
		from  int
		want  []block
	}{
		{
			// ケース 1: フラットなルート列
			//   兄弟 root のみ、子孫なし。順序が保持されること。
			name: "flat_roots",
			lines: []line{
				mk(0, kindUnchecked, "- [ ] todo1"),
				mk(0, kindChecked, "- [x] todo2"),
				mk(0, kindUnchecked, "- [ ] todo3"),
			},
			from: 0,
			want: []block{
				{head: mk(0, kindUnchecked, "- [ ] todo1")},
				{head: mk(0, kindChecked, "- [x] todo2")},
				{head: mk(0, kindUnchecked, "- [ ] todo3")},
			},
		},
		{
			// ケース 2: ネスト 2 段
			//   root → mid → leaf の三段ネストが正しく親子関係として畳まれること。
			name: "nested_two_levels",
			lines: []line{
				mk(0, kindUnchecked, "- [ ] root"),
				mk(1, kindUnchecked, "  - [ ] mid"),
				mk(2, kindChecked, "    - [x] leaf"),
			},
			from: 0,
			want: []block{
				{
					head: mk(0, kindUnchecked, "- [ ] root"),
					children: []block{
						{
							head: mk(1, kindUnchecked, "  - [ ] mid"),
							children: []block{
								{head: mk(2, kindChecked, "    - [x] leaf")},
							},
						},
					},
				},
			},
		},
		{
			// ケース 3: 兄弟と従兄弟混在
			//   - root a の下に 2 兄弟 (a1, a2) があり、a2 配下に孫 (a2-1) を持つ
			//   - root b の下に 1 子 (b1) のみ
			//   兄弟 (同 depth) と従兄弟 (異 root 配下の同 depth) の双方を 1 ケースで網羅する。
			name: "siblings_and_cousins_mix",
			lines: []line{
				mk(0, kindUnchecked, "- [ ] a"),
				mk(1, kindChecked, "  - [x] a1"),
				mk(1, kindUnchecked, "  - [ ] a2"),
				mk(2, kindChecked, "    - [x] a2-1"),
				mk(0, kindUnchecked, "- [ ] b"),
				mk(1, kindChecked, "  - [x] b1"),
			},
			from: 0,
			want: []block{
				{
					head: mk(0, kindUnchecked, "- [ ] a"),
					children: []block{
						{head: mk(1, kindChecked, "  - [x] a1")},
						{
							head: mk(1, kindUnchecked, "  - [ ] a2"),
							children: []block{
								{head: mk(2, kindChecked, "    - [x] a2-1")},
							},
						},
					},
				},
				{
					head: mk(0, kindUnchecked, "- [ ] b"),
					children: []block{
						{head: mk(1, kindChecked, "  - [x] b1")},
					},
				},
			},
		},
		{
			// ケース 4: kindOther ルート混在
			//   root 列にチェックボックスを持たない通常行 (memo / 空行など) が
			//   混入しても兄弟 root として扱われる。
			//   設計上「未完了パートの全行が網羅されること」(design.md L390-391) を担保する。
			name: "kindOther_root_mixed",
			lines: []line{
				mk(0, kindUnchecked, "- [ ] todo1"),
				mk(0, kindOther, "memo line"),
				mk(0, kindChecked, "- [x] todo2"),
			},
			from: 0,
			want: []block{
				{head: mk(0, kindUnchecked, "- [ ] todo1")},
				{head: mk(0, kindOther, "memo line")},
				{head: mk(0, kindChecked, "- [x] todo2")},
			},
		},
		{
			// 追加ケース: from > 0 で prefix を除外する
			//   findBoundary が返した境界以降のみ走査されること。
			//   prefix 部の行 (完了集側) は forest に含めない。
			name: "from_skips_prefix",
			lines: []line{
				mk(0, kindChecked, "- [x] done in completed section"),
				mk(0, kindOther, "## boundary"),
				mk(0, kindUnchecked, "- [ ] todo1"),
				mk(1, kindChecked, "  - [x] sub"),
			},
			from: 2,
			want: []block{
				{
					head: mk(0, kindUnchecked, "- [ ] todo1"),
					children: []block{
						{head: mk(1, kindChecked, "  - [x] sub")},
					},
				},
			},
		},
		{
			// 追加ケース: from が len(lines) と等しい (未完了パートなし) → nil
			//   findBoundary が len(lines) を返した場合の合成耐性。
			name:  "from_at_end_returns_nil",
			lines: []line{mk(0, kindChecked, "- [x] only done")},
			from:  1,
			want:  nil,
		},
		{
			// 追加ケース: from が負値 → nil (防御的)
			//   呼び出し側からの誤入力でも panic せず空フォレストを返すこと。
			name:  "from_negative_returns_nil",
			lines: []line{mk(0, kindUnchecked, "- [ ] todo")},
			from:  -1,
			want:  nil,
		},
		{
			// 追加ケース: kindOther 行が深い indent を持てば直前 root の子孫として吸収
			//   kind に依らず indent 関係のみで親子判定する不変条件 (design.md "走査と移動アルゴリズム" 3) を確認。
			name: "kindOther_with_indent_becomes_child",
			lines: []line{
				mk(0, kindUnchecked, "- [ ] todo1"),
				mk(1, kindOther, "  note about todo1"),
				mk(0, kindChecked, "- [x] todo2"),
			},
			from: 0,
			want: []block{
				{
					head: mk(0, kindUnchecked, "- [ ] todo1"),
					children: []block{
						{head: mk(1, kindOther, "  note about todo1")},
					},
				},
				{head: mk(0, kindChecked, "- [x] todo2")},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := buildBlocks(tc.lines, tc.from)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("buildBlocks mismatch (case=%s)\n got=%#v\nwant=%#v", tc.name, got, tc.want)
			}
		})
	}
}
