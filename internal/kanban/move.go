package kanban

import "strings"

// canMove はブロック b が完了集末尾へ移動可能かを判定する。
//
// 仕様 (tasks.md 2.4 / requirements.md R1.2, R3.3, R3.4, R3.5 /
//
//	design.md "内部補助関数（パッケージ非公開）" の canMove 行 +
//	"走査と移動アルゴリズム（不変条件）" 3):
//   - ルート b.head 自身が kindChecked であり、かつ
//     子孫を再帰的に走査して全ての kindChecked / kindUnchecked 行が kindChecked
//     (= kindUnchecked が 1 つも存在しない) ときのみ true を返す。
//   - ルートが kindUnchecked または kindOther の時点で false (R3.4 / R3.5 を構造的に吸収)。
//   - 子孫に kindUnchecked が 1 つでも残っていれば false (R3.3)。
//   - kindOther 子孫 (memo / 空行など) は判定中立で、true / false いずれにも寄与しない。
//
// 設計上の意図:
//   - canMove は「ルート単位で塊として移動できるか」のみを判定する。
//     子だけを抜き出して移動することは構造的に禁止されており (design.md 走査アルゴリズム 3)、
//     呼出側 (Sort 内のフォレスト走査) は root に対してのみ canMove を評価する。
//   - 純関数: ブロックを読むのみで副作用を持たない。time / rand / I/O への依存なし
//     (depguard / ast-grep の import 制約を遵守)。
func canMove(b block) bool {
	// ルートが kindChecked でなければ早期に false。
	// kindUnchecked: R3.4「親未完なら子完了でも単独移動禁止」を構造的に吸収。
	// kindOther:    そもそも移動対象外 (checkbox を持たない見出し / メモ等)。
	if b.head.kind != kindChecked {
		return false
	}
	// 子孫に kindUnchecked が 1 つでもあれば false (R3.3)。
	return !hasUncheckedDescendant(b.children)
}

// hasUncheckedDescendant は children 配下を再帰走査し、kindUnchecked 行が
// 1 つでも存在すれば true を返す。kindOther / kindChecked のみのときは false。
//
// canMove の補助関数。kind に対する判定の対称性のため独立させ、
// 「親が kindChecked でも子孫に [ ] があれば false」(R3.3) の不変条件を
// 単一の責務で表現する。
func hasUncheckedDescendant(children []block) bool {
	for _, c := range children {
		if c.head.kind == kindUnchecked {
			return true
		}
		if hasUncheckedDescendant(c.children) {
			return true
		}
	}
	return false
}

// flattenBlock はブロック b（root + 子孫）を木順序で line 配列に展開する。
//
// 仕様 (tasks.md 2.5 / design.md "走査と移動アルゴリズム"):
//   - head を先頭に置き、children を順序通りに再帰展開して連結する。
//   - 元の行順 (root → 第 1 子の subtree → 第 2 子の subtree → ...) を保持する。
//   - インデント文字種・幅は line.raw に保持されているため、本関数は値の変換を行わない。
//
// 純関数: ブロックを読むのみで副作用を持たない。assemble の補助。
func flattenBlock(b block) []line {
	out := []line{b.head}
	for _, c := range b.children {
		out = append(out, flattenBlock(c)...)
	}
	return out
}

// assemble は完了集 prefix・移動候補 completed・gap・未完了パート残留 remaining を
// この順で連結し、LF 統一文字列として再構築する。
//
// 仕様 (tasks.md 2.5 / design.md "走査と移動アルゴリズム" 4):
//   - prefix:    findInsertionIndex で確定した挿入位置までの行（完了集の最後の [x]
//     サブツリーまで）。raw をそのまま転送する。
//   - completed: 移動可ルートのフォレスト。flattenBlock で木順序に展開し、prefix の直後に追加する
//     （= 最後の完了ブロック直後への追記。R1.2「未完了パート内の出現順序を保ったまま、
//     完了集末尾へ追記」のうち「末尾」の解釈を「最後の [x] サブツリー直後」に固定する）。
//   - gap:       完了集側の挿入位置と未完了パート境界の間に挟まる kindOther 行（空行・
//     見出し等）。移動ブロック追記後にそのまま挿入し、remaining の直前に置く。
//   - remaining: 移動不可のルート群を flattenBlock 済みで連結した行列。
//     未完了パート内の元の出現順を保持する (R3.6)。
//
// 出力規約:
//   - 改行コード復元は呼び出し側 (fileio.AtomicWrite) の責務。本関数は LF 区切りのみ。
//   - 末尾改行は最終要素 (raw="") の存在で表現される。strings.Join でラウンドトリップ。
//
// 純関数: 引数を読むのみで副作用を持たない。strings.Join のみに依存。
func assemble(prefix []line, completed []block, gap []line, remaining []line) string {
	raws := make([]string, 0, len(prefix)+len(gap)+len(remaining))
	for _, l := range prefix {
		raws = append(raws, l.raw)
	}
	for _, b := range completed {
		for _, l := range flattenBlock(b) {
			raws = append(raws, l.raw)
		}
	}
	for _, l := range gap {
		raws = append(raws, l.raw)
	}
	for _, l := range remaining {
		raws = append(raws, l.raw)
	}
	return strings.Join(raws, "\n")
}
