---
paths:
  - "**/go.mod"
  - "**/package.json"
  - ".kiro/specs/**/*.md"
---
- spec / design に外部パッケージのバージョン下限・固定 version を契約として書く際、未リリースの将来バージョン (例: `vX.Y+` で X.Y が未公開) を指定してはならない。`@latest` で実測解決し go.mod / package.json に固定する手順に委譲する。
