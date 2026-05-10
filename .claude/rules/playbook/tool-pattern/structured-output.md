---
paths:
  - ".claude/commands/**/*.md"
  - ".claude/skills/**/*.md"
---
- slash command の runner-directive で「単一の json_schema object 形式で応答せよ」「AskUserQuestion 禁止」と指定されたら、StructuredOutput tool を呼び has_inquiry / summary / inquiry を持つ object を渡す。判断確定なら has_inquiry=false で summary、追加判断要なら true で inquiry.options[] を埋める。
