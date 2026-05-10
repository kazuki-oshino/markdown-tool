// task 3.1: Render の LCS ベース unified-diff 風出力に対するゴールデンテスト。
// design.md "Pure View / internal/diffview" の Service Interface 契約を固定する。
package diffview

import (
	"embed"
	"testing"
)

// goldenFS は internal/diffview/testdata/ 配下のゴールデンフィクスチャ
// (`*.before.md`, `*.after.md`, `*.diff`) を compile-time に埋め込む。
// kanban パッケージと同じ慣行 (パッケージ直下 testdata/ + go:embed) で揃える。
//
//go:embed testdata/*.before.md testdata/*.after.md testdata/*.diff
var goldenFS embed.FS

// goldenCases は tasks.md 3.1 が要求する代表 4 ケースを網羅する:
//   - 移動 1 件        : single_move
//   - 移動複数件      : multiple_moves
//   - no-op (空文字列): no_op
//   - 親子ブロック移動: parent_child_move
var goldenCases = []string{
	"single_move",
	"multiple_moves",
	"no_op",
	"parent_child_move",
}

// TestRender_Goldens は testdata/*.before.md と testdata/*.after.md を Render に通した
// 結果が testdata/*.diff と完全一致することを確認する。
// design.md "Pure View / Service Interface" の Postconditions（unified-diff 風 +/-/space
// プレフィックス + @@ ハンクヘッダ）と Invariants（before==after で空文字列）を
// 構造的に検証する。
func TestRender_Goldens(t *testing.T) {
	t.Parallel()

	for _, base := range goldenCases {
		t.Run(base, func(t *testing.T) {
			t.Parallel()

			beforeBytes, err := goldenFS.ReadFile("testdata/" + base + ".before.md")
			if err != nil {
				t.Fatalf("read %s.before.md: %v", base, err)
			}
			afterBytes, err := goldenFS.ReadFile("testdata/" + base + ".after.md")
			if err != nil {
				t.Fatalf("read %s.after.md: %v", base, err)
			}
			diffBytes, err := goldenFS.ReadFile("testdata/" + base + ".diff")
			if err != nil {
				t.Fatalf("read %s.diff: %v", base, err)
			}

			got := Render(string(beforeBytes), string(afterBytes))
			want := string(diffBytes)
			if got != want {
				t.Fatalf("Render(%s) mismatch\n--- got (%d bytes) ---\n%q\n--- want (%d bytes) ---\n%q",
					base, len(got), got, len(want), want)
			}
		})
	}
}

// TestRender_EqualReturnsEmpty は before == after なら空文字列を返す不変条件
// （design.md "Pure View / Invariants" 第 1 項）を独立に確認する。
func TestRender_EqualReturnsEmpty(t *testing.T) {
	t.Parallel()

	cases := []string{
		"",
		"abc\n",
		"abc\ndef\nghi\n",
		"末尾改行なし",
	}
	for _, in := range cases {
		if got := Render(in, in); got != "" {
			t.Errorf("Render(%q, %q) = %q; want \"\"", in, in, got)
		}
	}
}

// TestRender_Determinism は同一入力に対し同一出力を返す決定性
// （design.md "Pure View / Invariants" 第 2 項）を簡易に確認する。
func TestRender_Determinism(t *testing.T) {
	t.Parallel()

	beforeBytes, err := goldenFS.ReadFile("testdata/multiple_moves.before.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	afterBytes, err := goldenFS.ReadFile("testdata/multiple_moves.after.md")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	before := string(beforeBytes)
	after := string(afterBytes)

	first := Render(before, after)
	for i := range 5 {
		again := Render(before, after)
		if again != first {
			t.Fatalf("Render not deterministic at iteration %d\n--- first ---\n%q\n--- again ---\n%q",
				i, first, again)
		}
	}
}
