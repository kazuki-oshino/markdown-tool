---
paths:
  - "**/*.go"
  - "**/*_test.go"
---
- `reflect.DeepEqual` は子 slice の `nil` と `[]T{}` を等価判定しない。再帰木構造で `children []T` を比較する場合は、実装側でも expected 側でも空時は append せず nil のまま残す方針で揃える。`if len(...) > 0 { result = append(...) }` ではなく append 後に nil 確認する。
- line-based 文字列変換 (parse → 加工 → assemble) の round-trip を保つには parse を `strings.Split(input, "\n")`、assemble を `strings.Join(raws, "\n")` に固定する。空入力でも Split の戻り `[""]` を 1 行扱いにすれば `""` / `"foo"` / `"foo\n"` の三系統が改行コード変換なしで round-trip する。
