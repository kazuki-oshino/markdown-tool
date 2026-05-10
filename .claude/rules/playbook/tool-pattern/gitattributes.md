---
paths:
  - ".gitattributes"
  - "testdata/**"
---
- `.gitattributes` のルール追加直後に `git check-attr -a -- <included-path> <excluded-path>` を 1 コマンドで両方渡し、included 側に期待属性 (例: `text: unset`) が出力され excluded 側には対象属性が出力されない (空行) ことを実測してから commit する。包含のみ確認すると兄弟 dir への誤拡散を見逃す。
