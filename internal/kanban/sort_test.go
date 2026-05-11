// task 2.5: Sort 公開 API のゴールデンテスト + 冪等性テスト。
// design.md "Pure Domain / internal/kanban" の Service Interface 契約を固定する。
package kanban

import (
	"embed"
	"strings"
	"testing"
)

// goldenFS は internal/kanban/testdata/ 配下の LF 統一フィクスチャを compile-time に
// 埋め込む。embed パッケージは depguard の禁止リスト（os / io / io/fs / time 等）に
// 含まれず、純ドメインの import 制約を破らずにテスト用フィクスチャを参照できる
// 唯一の経路として採用する。
//
// fixtures はリポジトリ直下 testdata/kanban/ ではなく Go 標準慣行に従って
// パッケージ内 testdata/ に配置している（embed の親ディレクトリ参照不可制約により
// 必然。design.md File Structure Plan の repo-root testdata/ は fileio 用に維持）。
//
//go:embed testdata/*.md.in testdata/*.md.out
var goldenFS embed.FS

// goldenCases は testdata/ 配下のフィクスチャ basename 列。
// task 1.3 で配置された 5 ペア（basic_unchanged / single_x_move /
// parent_child_all_done / parent_done_child_open / child_done_parent_open）を
// 全件カバーする (tasks.md 2.5 の「全件 pass」要件)。
var goldenCases = []string{
	"basic_unchanged",
	"single_x_move",
	"parent_child_all_done",
	"parent_done_child_open",
	"child_done_parent_open",
	"blank_gap_after_completed",
	"divider_completed_below",
	"multiple_dividers_last_wins",
}

// TestSort_Goldens は testdata/*.md.in を Sort に通した結果が
// 対応する testdata/*.md.out と完全一致することを確認する。
// design.md "Pure Domain" の Postconditions（順序保持・インデント保存・LF 統一）と
// requirements.md R1.1, R1.2, R1.3, R2.2, R2.3, R3.5, R3.6 の構造的保証を
// ゴールデン入出力で総当たり的に検証する。
func TestSort_Goldens(t *testing.T) {
	t.Parallel()

	for _, base := range goldenCases {
		t.Run(base, func(t *testing.T) {
			t.Parallel()

			inBytes, err := goldenFS.ReadFile("testdata/" + base + ".md.in")
			if err != nil {
				t.Fatalf("read %s.md.in: %v", base, err)
			}
			outBytes, err := goldenFS.ReadFile("testdata/" + base + ".md.out")
			if err != nil {
				t.Fatalf("read %s.md.out: %v", base, err)
			}

			input := string(inBytes)
			want := string(outBytes)

			// kanban 層フィクスチャは LF 統一前提 (tasks.md 1.3, 2.5 /
			// design.md "改行コード責務の分割")。CRLF 混入は責務分割違反を意味するため
			// テスト時点で検出して即座に失敗させる。
			if strings.Contains(input, "\r") {
				t.Fatalf("kanban fixture %s.md.in contains CR (CRLF); kanban fixtures must be LF-only", base)
			}
			if strings.Contains(want, "\r") {
				t.Fatalf("kanban fixture %s.md.out contains CR (CRLF); kanban fixtures must be LF-only", base)
			}

			got, err := Sort(input)
			if err != nil {
				t.Fatalf("Sort(%s) returned err: %v", base, err)
			}
			if got != want {
				t.Fatalf("Sort(%s) mismatch\n--- got ---\n%q\n--- want ---\n%q", base, got, want)
			}
		})
	}
}

// TestSort_Idempotent は Sort(Sort(x)) == Sort(x) をすべてのゴールデンケースで
// 確認する。design.md "Pure Domain / Invariants" 第 2 項の冪等性契約に対応。
//
// 設計上の意図:
//   - 1 回目で完了集末尾へ移動された [x] ブロックは、2 回目の boundary 走査では
//     prefix（完了集側）に含まれており、buildBlocks の対象外となるため再移動されない。
//   - 同一入力に対し常に同一出力を返す決定性 (R7.2) と直交する不変条件として
//     冪等性を独立に検証する。
func TestSort_Idempotent(t *testing.T) {
	t.Parallel()

	for _, base := range goldenCases {
		t.Run(base, func(t *testing.T) {
			t.Parallel()

			inBytes, err := goldenFS.ReadFile("testdata/" + base + ".md.in")
			if err != nil {
				t.Fatalf("read %s.md.in: %v", base, err)
			}
			input := string(inBytes)

			once, err := Sort(input)
			if err != nil {
				t.Fatalf("Sort(%s) returned err: %v", base, err)
			}
			twice, err := Sort(once)
			if err != nil {
				t.Fatalf("Sort(Sort(%s)) returned err: %v", base, err)
			}
			if twice != once {
				t.Fatalf("Sort not idempotent for %s\n--- Sort(x) ---\n%q\n--- Sort(Sort(x)) ---\n%q", base, once, twice)
			}
		})
	}
}
