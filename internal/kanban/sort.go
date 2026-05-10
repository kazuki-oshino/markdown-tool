package kanban

// Sort は LF 統一済み Markdown 文字列を入力に、未完了パート内の移動可能な
// [x] ブロックを完了集末尾へ移動した結果を返す決定的純関数。
//
// 仕様 (tasks.md 2.5 / design.md "Pure Domain / internal/kanban" の Service Interface /
//
//	requirements.md R1.1, R1.2, R1.3, R2.2, R2.3, R3.5, R3.6, R7.1, R7.2):
//
//   - 入力 input は LF (\n) 統一前提。CRLF 正規化は呼び出し側 (fileio.Read) の責務。
//   - 入力 == 出力（移動対象なし／未完了パートなし／完了集空）も等価文字列を返す（早期リターン）。
//   - 移動対象ブロックの内部行順序・インデント文字種・インデント幅は保持する。
//   - 同一入力に対し常に同一出力を返す（決定性）。Sort(Sort(x)) == Sort(x)（冪等性）。
//
// アルゴリズム (design.md "走査と移動アルゴリズム"):
//  1. parseLines で line 配列に分解
//  2. findBoundary で完了集末尾位置 b を取得
//     - b == len(lines): [ ] が一切なし → 入力をそのまま返す (R2.2)
//     - b == 0:          先頭が [ ] → 完了集が空。移動対象も存在しないため
//       ファイル不変として早期リターン (R2.3)
//  3. buildBlocks(lines, b) で未完了パート全体を順序付きフォレストに展開
//  4. 各ルートに canMove を適用し、true なら completed、false なら remaining
//     （子だけを抜き出すことは禁止: R3.4 を構造的に保証）
//  5. completed が空なら入力不変として早期リターン (R1.3)
//  6. assemble(prefix, completed, remaining) で LF 統一文字列を再構築
//
// error は将来の致命的不整合（UTF-8 不正等）専用で、本実装では常に nil を返す
// （design.md "Service Interface" Preconditions / Implementation Notes の MVP 方針）。
func Sort(input string) (string, error) {
	lines := parseLines(input)
	boundary := findBoundary(lines)

	// R2.2: [ ] が一切存在しない → 未完了パートなし、ファイル不変。
	if boundary == len(lines) {
		return input, nil
	}
	// R2.3: 先頭が [ ] → 完了集が空。移動対象も存在しないため不変。
	if boundary == 0 {
		return input, nil
	}

	forest := buildBlocks(lines, boundary)

	var completed []block
	var remaining []line
	for _, root := range forest {
		if canMove(root) {
			completed = append(completed, root)
			continue
		}
		// 移動不可のルートはサブツリー全体をそのまま remaining に展開する。
		// 子だけを抜き出して移動することは設計上禁止 (R3.4)。
		remaining = append(remaining, flattenBlock(root)...)
	}

	// R1.3: 移動対象が 1 件もなければ入力不変。
	if len(completed) == 0 {
		return input, nil
	}

	return assemble(lines[:boundary], completed, remaining), nil
}
