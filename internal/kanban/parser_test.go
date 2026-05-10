// task 2.1: parser.go の indentDepth / parseLines に対するテーブルテスト。
// design.md "インデント混在の正規化規則" / "Domain Model（純ロジック内部の値型）" を契約として固定する。
package kanban

import "testing"

// TestIndentDepth は design.md L399-402 の正規化規則をテーブルで網羅する。
//   - タブ 1 個 = 深さ +1
//   - 半角スペース 2 個ごとに +1（端数 1 個は加算しない）
//   - 半角スペース 1 個は深さ 0
//   - タブと半角の混在: タブ優先解釈。タブ後に続く半角は同一段の継続として深さに加えない。
//
// tasks.md 2.1 で要求された 5 系統「タブ単独」「半角 2」「半角 4」「半角 1（無効）」「タブ+半角混在」を含み、
// 端数 (半角 3) や複数段（半角→タブ）等の境界も決定論で固定する。
func TestIndentDepth(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int
	}{
		// タブ単独
		{"empty string", "", 0},
		{"tab only line", "\t", 1},
		{"single tab + content", "\thello", 1},
		{"two tabs + content", "\t\thello", 2},
		{"three tabs + content", "\t\t\thello", 3},

		// 半角 2 / 4 / 半角 1（無効）
		{"no leading whitespace", "hello", 0},
		{"single space (invalid, depth 0)", " hello", 0},
		{"two spaces", "  hello", 1},
		{"three spaces (fraction not added)", "   hello", 1},
		{"four spaces", "    hello", 2},
		{"five spaces", "     hello", 2},
		{"six spaces", "      hello", 3},

		// タブ+半角混在
		{"tab + 1 space (continuation)", "\t hello", 1},
		{"tab + 2 spaces (continuation)", "\t  hello", 1},
		{"tab + 4 spaces (continuation)", "\t    hello", 1},
		{"tab + tab", "\t\thello", 2},
		{"two spaces + tab (spaces precede tab)", "  \thello", 2},
		{"two tabs + 2 spaces (continuation)", "\t\t  hello", 2},

		// 非空白文字到達で走査終了 (空白以降の検査は影響しない)
		{"tab inside content does not count", "x\thello", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := indentDepth(tc.raw); got != tc.want {
				t.Fatalf("indentDepth(%q) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}

// TestParseLines は parseLines が LF 分割で各行を line に変換し、
// indent / kind / raw の不変条件を保つことを検証する。
//
// 不変条件:
//   - raw は LF 分割後の素の文字列（改行を含まない）
//   - indent は indentDepth(raw) と一致
//   - kind: "[x]" or "[X]" を含めば kindChecked / "[ ]" を含めば kindUnchecked / それ以外は kindOther
//   - 末尾改行は最終要素として空行 (raw="") を生成し、strings.Join("\n") で round-trip 可能
func TestParseLines(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []line
	}{
		{
			name:  "empty input yields single empty line",
			input: "",
			want: []line{
				{indent: 0, kind: kindOther, raw: ""},
			},
		},
		{
			name:  "single unchecked task no indent",
			input: "- [ ] task A",
			want: []line{
				{indent: 0, kind: kindUnchecked, raw: "- [ ] task A"},
			},
		},
		{
			name:  "single checked task lowercase x",
			input: "- [x] task A",
			want: []line{
				{indent: 0, kind: kindChecked, raw: "- [x] task A"},
			},
		},
		{
			name:  "single checked task uppercase X",
			input: "- [X] task A",
			want: []line{
				{indent: 0, kind: kindChecked, raw: "- [X] task A"},
			},
		},
		{
			name:  "plain header line is kindOther",
			input: "# Heading",
			want: []line{
				{indent: 0, kind: kindOther, raw: "# Heading"},
			},
		},
		{
			name:  "blank line is kindOther",
			input: " ",
			want: []line{
				{indent: 0, kind: kindOther, raw: " "},
			},
		},
		{
			name:  "tab-indented checked",
			input: "\t- [x] child",
			want: []line{
				{indent: 1, kind: kindChecked, raw: "\t- [x] child"},
			},
		},
		{
			name:  "two-space-indented unchecked",
			input: "  - [ ] child",
			want: []line{
				{indent: 1, kind: kindUnchecked, raw: "  - [ ] child"},
			},
		},
		{
			name:  "four-space-indented checked is depth 2",
			input: "    - [x] grandchild",
			want: []line{
				{indent: 2, kind: kindChecked, raw: "    - [x] grandchild"},
			},
		},
		{
			name:  "tab + 2 spaces + checked (continuation, depth 1)",
			input: "\t  - [x] tab+spaces",
			want: []line{
				{indent: 1, kind: kindChecked, raw: "\t  - [x] tab+spaces"},
			},
		},
		{
			name:  "multi-line mix preserves order and round-trips via strings.Join",
			input: "- [x] A\n  - [x] A1\n- [ ] B",
			want: []line{
				{indent: 0, kind: kindChecked, raw: "- [x] A"},
				{indent: 1, kind: kindChecked, raw: "  - [x] A1"},
				{indent: 0, kind: kindUnchecked, raw: "- [ ] B"},
			},
		},
		{
			name:  "trailing LF produces empty last line for round-trip",
			input: "- [x] A\n",
			want: []line{
				{indent: 0, kind: kindChecked, raw: "- [x] A"},
				{indent: 0, kind: kindOther, raw: ""},
			},
		},
		{
			name:  "header + blank + checked + unchecked composite",
			input: "# Done\n- [x] A\n\n# Todo\n- [ ] B",
			want: []line{
				{indent: 0, kind: kindOther, raw: "# Done"},
				{indent: 0, kind: kindChecked, raw: "- [x] A"},
				{indent: 0, kind: kindOther, raw: ""},
				{indent: 0, kind: kindOther, raw: "# Todo"},
				{indent: 0, kind: kindUnchecked, raw: "- [ ] B"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseLines(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("parseLines(%q) len = %d, want %d (got=%+v)", tc.input, len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("parseLines(%q)[%d] = %+v, want %+v", tc.input, i, got[i], tc.want[i])
				}
			}
		})
	}
}
