// task 2.4: move.go の canMove に対するテーブルテスト。
// design.md "走査と移動アルゴリズム（不変条件）" / "内部補助関数（パッケージ非公開）" の
// canMove 行を契約として固定する。
package kanban

import "testing"

// TestCanMove は canMove(b block) が以下の真偽値仕様を満たすことを検証する。
//
// 仕様 (tasks.md 2.4 / requirements.md R1.2, R3.3, R3.4, R3.5 /
//
//	design.md "内部補助関数" canMove 行 + "走査と移動アルゴリズム"):
//   - ルート自身が kindChecked かつ子孫の全 kindChecked / kindUnchecked 行が
//     kindChecked であるとき true。
//   - ルートが kindUnchecked または kindOther のときは false (R3.4 親未完だと子完了でも単独移動禁止 / kindOther はそもそも移動対象外)。
//   - 子孫に kindUnchecked が 1 つでも残れば false (R3.3)。
//   - kindOther の子孫 (memo / 空行 等) は判定に影響しない (kind は [x] / [ ] のみが対象)。
//
// 必須ケース (tasks.md 2.4):
//  1. 親子全完了 → true
//  2. 親完了 / 子未完 → false (R3.3)
//  3. 子完了 / 親未完 → false (R3.4)
//  4. ルートが kindOther → false
//  5. 単独 [x] ルート (子孫なし) → true
//
// 追加ケース:
//  6. 孫まで全て [x] → true
//  7. 深いネストのうち孫に [ ] が残る → false
//  8. 子孫に kindOther が混在しても [x]/[ ] のみで判定される → true
//  9. ルートが kindUnchecked かつ子孫なし → false (5 と対称な未完了ルート単独)
func TestCanMove(t *testing.T) {
	t.Parallel()

	// テスト用 block 構築ヘルパ。indent / raw は canMove が参照しないため
	// 可読性のためのプレースホルダ値で良い。canMove は kind のみを再帰的に参照する。
	mk := func(k kind, raw string) line {
		return line{indent: 0, kind: k, raw: raw}
	}

	cases := []struct {
		name string
		b    block
		want bool
	}{
		{
			// ケース 1: 親子全完了
			//   root [x] + child [x] + grandchild [x] → true
			//   tasks.md 2.4 必須ケース「親子全完了」。
			name: "all_done_parent_and_children",
			b: block{
				head: mk(kindChecked, "- [x] root"),
				children: []block{
					{head: mk(kindChecked, "  - [x] child")},
				},
			},
			want: true,
		},
		{
			// ケース 2: 親完了 / 子未完 (R3.3)
			//   root [x] だが子に [ ] が残る → false。
			//   親 [x] の子孫に [ ] が 1 つでも残れば移動禁止という不変条件を確認。
			name: "parent_done_child_open",
			b: block{
				head: mk(kindChecked, "- [x] root"),
				children: []block{
					{head: mk(kindUnchecked, "  - [ ] child")},
				},
			},
			want: false,
		},
		{
			// ケース 3: 子完了 / 親未完 (R3.4)
			//   root [ ] かつ子に [x] が居る → false。
			//   子 [x] が居ても親が [ ] なら単独移動禁止 (構造的な R3.4 吸収)。
			name: "child_done_parent_open",
			b: block{
				head: mk(kindUnchecked, "- [ ] root"),
				children: []block{
					{head: mk(kindChecked, "  - [x] child")},
				},
			},
			want: false,
		},
		{
			// ケース 4: ルートが kindOther
			//   見出し / 空行 / プレーンテキスト等、checkbox を持たない root は
			//   そもそも移動対象外 → false。
			//   tasks.md 2.4 必須ケース「ルートが kindOther」。
			name: "root_is_kindOther",
			b: block{
				head: mk(kindOther, "## 見出し"),
			},
			want: false,
		},
		{
			// ケース 5: 単独 [x] ルート (子孫なし)
			//   children が nil の最も単純な完了 root。
			//   tasks.md 2.4 必須ケース「単独 [x] ルート」。
			name: "single_checked_root_no_children",
			b: block{
				head: mk(kindChecked, "- [x] standalone"),
			},
			want: true,
		},
		{
			// 追加ケース 6: 孫まで全て [x]
			//   3 段ネストの全 [x] が true を返すこと（深さの再帰走査が成立する確認）。
			name: "deep_nested_all_done",
			b: block{
				head: mk(kindChecked, "- [x] root"),
				children: []block{
					{
						head: mk(kindChecked, "  - [x] child"),
						children: []block{
							{head: mk(kindChecked, "    - [x] grandchild")},
						},
					},
				},
			},
			want: true,
		},
		{
			// 追加ケース 7: 深いネストのうち孫に [ ] が残る
			//   root [x] / child [x] / grandchild [ ] → false。
			//   子孫の任意の深さの [ ] が false を引き起こすこと（再帰の網羅性）。
			name: "deep_nested_grandchild_open",
			b: block{
				head: mk(kindChecked, "- [x] root"),
				children: []block{
					{
						head: mk(kindChecked, "  - [x] child"),
						children: []block{
							{head: mk(kindUnchecked, "    - [ ] grandchild")},
						},
					},
				},
			},
			want: false,
		},
		{
			// 追加ケース 8: kindOther の子孫が混在しても [x]/[ ] のみで判定
			//   root [x] + child kindOther (memo) + child [x] → true。
			//   design.md "canMove" 行: 「子孫の全 [x]/[ ] 行が kindChecked」が満たされること。
			//   kindOther 子孫は判定中立 (true 側でも false 側でもない、無視される)。
			name: "kindOther_descendant_is_neutral",
			b: block{
				head: mk(kindChecked, "- [x] root"),
				children: []block{
					{head: mk(kindOther, "  memo about root")},
					{head: mk(kindChecked, "  - [x] child")},
				},
			},
			want: true,
		},
		{
			// 追加ケース 9: ルートが kindUnchecked かつ子孫なし
			//   ケース 5 と対称な「ルート単独 [ ]」。
			//   ルートが [x] でない時点で false（早期判定の確認）。
			name: "single_unchecked_root_no_children",
			b: block{
				head: mk(kindUnchecked, "- [ ] standalone"),
			},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := canMove(tc.b)
			if got != tc.want {
				t.Errorf("canMove(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}
