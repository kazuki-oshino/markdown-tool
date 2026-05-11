package kanban

// block は parent + descendants の順序付き木ノード。
//
// フィールド:
//   - head:     ブロックの先頭行 (root 行 or 子孫行)
//   - children: head より深い indent を持つ直接の子ブロック列 (順序付き)
//
// 不変条件 (design.md "Domain Model（純ロジック内部の値型）"):
//   - children に含まれる各 block の head.indent > 親の head.indent
//   - children は元の行順を保持する
//   - 子・孫・ひ孫... と再帰的に同じ構造を取る (children も []block)
type block struct {
	head     line
	children []block
}

// buildBlocks は lines[from:] を「未完了パートの全行」とみなし、
// 順序付きフォレストとして構築する。
//
// 仕様 (tasks.md 2.3 / requirements.md R3.2 / design.md "走査と移動アルゴリズム"):
//   - lines[from:] の全行 (kindOther / kindUnchecked / kindChecked / kindDivider) を
//     フォレストの要素 (root or descendant) として網羅する。
//   - 連続する child.indent > parent.indent 行を直前 root の子孫範囲として吸収する。
//   - 子孫範囲内では再帰的に同じルールでサブフォレストを構築する。
//   - 元の行順 (root 順 / 各 children 順) を保持する。
//
// 設計上のトレードオフ:
//   - design.md "走査と移動アルゴリズム" 2 では「最浅インデント行をルート」と表現するが、
//     実装は「先頭行を root 候補として直後の連続 deeper 行を子孫に吸収する貪欲アルゴリズム」を採用する。
//     典型的な kanban (root 行が最浅) では両定義は等価であり、先頭行が後続より深い
//     稀ケースでも全行網羅の不変条件 (design.md L390-391) を優先して "先頭行も root" として扱う。
//
// エッジケース:
//   - from < 0 または from >= len(lines) → nil を返す
//     (findBoundary が len(lines) を返した「未完了パートなし」ケースを安全に吸収する)
//
// 純関数: lines を読むのみで副作用を持たない (depguard / ast-grep で機械検証される import 制約に従う)。
func buildBlocks(lines []line, from int) []block {
	if from < 0 || from >= len(lines) {
		return nil
	}
	return buildForest(lines[from:])
}

// buildForest は s 内の各先頭行を root 候補として、それより深い indent の連続を
// 子孫範囲として吸収するアルゴリズムでフォレストを構築する再帰ヘルパ。
//
// アルゴリズム:
//  1. i 番目の行を新しい block の head とする (currentIndent = head.indent)。
//  2. j を i+1 から進め、s[j].indent > currentIndent の間は子孫範囲。
//  3. j が停止した時点で s[i+1:j] を子孫として再帰的に buildForest し、
//     その結果を block.children として束ねる (子孫範囲が空なら children は nil)。
//  4. i を j に進め、s 末尾まで繰り返す。
//
// 不変条件:
//   - 戻り値 forest 内の隣接 root は indent の大小関係に依存せず、
//     子孫範囲の連続性 (indent > root.indent) のみで分割される。
//   - children は元の行順を保ったまま再帰的に同じ構造を取る。
//   - kind には依存せず indent 関係のみで親子判定する
//     (kindOther 行も deeper indent なら子として吸収される)。
func buildForest(s []line) []block {
	var blocks []block
	i := 0
	for i < len(s) {
		head := s[i]
		// 子孫範囲: head 直後で head.indent より strict に深い行が連続する範囲
		j := i + 1
		for j < len(s) && s[j].indent > head.indent {
			j++
		}
		b := block{head: head}
		if j > i+1 {
			b.children = buildForest(s[i+1 : j])
		}
		blocks = append(blocks, b)
		i = j
	}
	return blocks
}
