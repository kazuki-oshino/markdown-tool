---
paths:
  - "cmd/**/*.go"
---
- cobra で `SetUsageFunc` を上書きしその中で `c.UsageString()` を呼ぶと UsageString → Usage → UsageFunc の無限再帰になる。SetUsageFunc は使わず Args / SetFlagErrorFunc 内で `fmt.Fprint(cmd.ErrOrStderr(), cmd.UsageString())` を直接呼ぶ。`OutOrStderr()` は SetOut の writer を返すため stderr 出力には `ErrOrStderr()` を明示する。
