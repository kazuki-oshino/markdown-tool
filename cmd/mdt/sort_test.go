// cmd/mdt の sort サブコマンドに対するユニットテスト。
//
// 検証範囲 (tasks.md 6.2 / requirements.md R1.1, R1.3, R1.5, R4.1, R4.3, R4.6,
// R5.1, R5.3, R5.4):
//   - 通常実行: kanban.Sort により [x] が完了集末尾へ移動し、対象ファイルが
//     アトミックに上書きされ、終了コード 0 を返すこと (R1.1, R4.3)
//   - --dry-run: 差分が stdout に出力され、対象ファイルは変更されないこと
//     (R4.1, R4.6)
//   - no-op: 入力 == 出力なら書き込みも diff 出力もせず終了コード 0 を返すこと
//     (R1.3, R2.2)
//   - 引数省略: cobra.ExactArgs バリデーション失敗で UsageError 分類のエラーを返し、
//     使用方法ヒントを stderr に出すこと (R5.4)
//   - 未知フラグ: FlagErrorFunc 経由で UsageError 分類のエラーを返すこと (R5.3)
//   - 存在しないファイル: fileio.Read 失敗が ErrNotExist を保つ runtime error として
//     伝搬し、UsageError 分類にはならないこと (R1.5)
//
// 設計上の補足:
//   - Cobra の OutOrStderr() は SetOut が設定されているとそちら (=テストの stdout バッファ)
//     を返すため、使用方法は ErrOrStderr() に明示的に書き出す方針。テストはその契約を
//     stderr バッファ上で検証する (cmd/mdt/sort.go の SetUsageFunc / Args 実装と整合)。
package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readFile はテスト用のファイル読み込みヘルパ。失敗時は t.Fatal で停止する。
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ファイル読み込み失敗 %q: %v", path, err)
	}
	return string(b)
}

// writeFile はテスト用のファイル書き込みヘルパ。失敗時は t.Fatal で停止する。
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("ファイル書き込み失敗 %q: %v", path, err)
	}
}

// TestSort_NormalWrite_UpdatesFile は `mdt sort <file>` がファイルをアトミック上書きし、
// 完了行が完了集末尾へ移動した結果が反映されることを確認する。
// 対応要件: R1.1 (移動)、R4.3 (上書き)、R5.1 (位置引数)。
func TestSort_NormalWrite_UpdatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kanban.md")
	// 完了集 [x] A の後ろに未完了パートが続き、その中の [x] C が移動対象。
	input := "- [x] A\n- [ ] B\n- [x] C\n"
	writeFile(t, path, input)

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"sort", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}

	got := readFile(t, path)
	want := "- [x] A\n- [x] C\n- [ ] B\n"
	if got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
	if stdout.Len() != 0 {
		t.Errorf("通常書き込みでは stdout は空であるべき: %q", stdout.String())
	}
}

// TestSort_DryRun_DiffOnStdout は `mdt sort --dry-run <file>` が差分を stdout に出し、
// ファイルを変更しないことを確認する。
// 対応要件: R4.1 (--dry-run の差分プレビュー)、R4.6 (--dry-run でファイル不変)。
func TestSort_DryRun_DiffOnStdout(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kanban.md")
	input := "- [x] A\n- [ ] B\n- [x] C\n"
	writeFile(t, path, input)

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"sort", "--dry-run", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}

	if got := readFile(t, path); got != input {
		t.Errorf("--dry-run 中にファイルが変更された: got %q, want %q", got, input)
	}
	out := stdout.String()
	// LCS の出力規約: 移動行は元位置に "-", 移動先に "+" として両方出現する (R4.2)。
	if !strings.Contains(out, "-- [x] C") {
		t.Errorf("stdout に移動元 (- [x] C) の削除マーカーが含まれない: %q", out)
	}
	if !strings.Contains(out, "+- [x] C") {
		t.Errorf("stdout に移動先 (- [x] C) の追加マーカーが含まれない: %q", out)
	}
}

// TestSort_NoOp_FileUnchanged は移動対象が無い入力でファイル不変・stdout 空・終了コード 0 を確認する。
// 対応要件: R1.3 (移動対象なしならファイル不変)、R2.2 ([ ] が一切ない場合)。
func TestSort_NoOp_FileUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kanban.md")
	// [ ] が一切ない → kanban.Sort は早期リターンで input をそのまま返す。
	input := "- [x] A\n- [x] B\n"
	writeFile(t, path, input)

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"sort", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}

	if got := readFile(t, path); got != input {
		t.Errorf("no-op でファイルが変更された: got %q, want %q", got, input)
	}
	if stdout.Len() != 0 {
		t.Errorf("no-op では stdout は空であるべき: %q", stdout.String())
	}
}

// TestSort_DryRunNoOp_NoOutput は --dry-run + no-op 入力で stdout が空、ファイル不変を確認する。
// 設計フロー (design.md sequence) に従い content == sorted は --dry-run より前に no-op 判定する。
func TestSort_DryRunNoOp_NoOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kanban.md")
	input := "- [x] A\n- [x] B\n"
	writeFile(t, path, input)

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"sort", "--dry-run", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}

	if got := readFile(t, path); got != input {
		t.Errorf("--dry-run no-op でファイルが変更された: got %q, want %q", got, input)
	}
	if stdout.Len() != 0 {
		t.Errorf("--dry-run no-op では stdout は空であるべき: %q", stdout.String())
	}
}

// TestSort_MissingArg_UsageError は引数省略時に UsageError 分類のエラーが返り、
// stderr に使用方法ヒントが出ることを確認する。
// 対応要件: R5.4 (使用方法ヒント付きエラー + 非ゼロ終了コード)。
func TestSort_MissingArg_UsageError(t *testing.T) {
	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"sort"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("引数省略でエラーが返らなかった")
	}
	if !isUsageError(err) {
		t.Errorf("err = %v, want classifiable as usage error", err)
	}
	// 使用方法ヒントは ErrOrStderr() (= テストの stderr バッファ) に書き出す契約。
	if !strings.Contains(stderr.String(), "Usage:") {
		t.Errorf("stderr に \"Usage:\" が含まれない: %q", stderr.String())
	}
}

// TestSort_TooManyArgs_UsageError は引数過多時にも UsageError 分類のエラーが返ることを確認する。
// 対応要件: R5.3 (使用方法不正は非ゼロ終了)。cobra.ExactArgs(1) の契約を構造的に固定する。
func TestSort_TooManyArgs_UsageError(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"sort", "a.md", "b.md"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("引数過多でエラーが返らなかった")
	}
	if !isUsageError(err) {
		t.Errorf("err = %v, want classifiable as usage error", err)
	}
}

// TestSort_UnknownFlag_UsageError は未知フラグで UsageError 分類のエラーが返ることを確認する。
// 対応要件: R5.3 (使用方法不正)。
func TestSort_UnknownFlag_UsageError(t *testing.T) {
	cmd := newRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"sort", "--no-such-flag", "anything.md"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("未知フラグでエラーが返らなかった")
	}
	if !isUsageError(err) {
		t.Errorf("err = %v, want classifiable as usage error", err)
	}
}

// TestSort_NonExistentFile_RuntimeError は存在しないファイルを指定した場合に
// runtime error として伝搬し、UsageError 分類にはならず、ErrNotExist を保つことを確認する。
// 対応要件: R1.5 (読み取り失敗時の失敗理由付きエラー + 非ゼロ終了コード)。
func TestSort_NonExistentFile_RuntimeError(t *testing.T) {
	nonExist := filepath.Join(t.TempDir(), "does_not_exist.md")

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"sort", nonExist})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("存在しないファイルでエラーが返らなかった")
	}
	if isUsageError(err) {
		t.Errorf("err は runtime error 期待だが usage error に分類された: %v", err)
	}
	// fileio.Read は %w で wrap しているため errors.Is で os.ErrNotExist を判定可能。
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want wraps os.ErrNotExist", err)
	}
	// classifyExitCode が 1 (ExitRuntimeError) を返すことも構造的に固定する。
	if got := classifyExitCode(err); got != ExitRuntimeError {
		t.Errorf("classifyExitCode = %d, want %d", got, ExitRuntimeError)
	}
}

// TestSort_PreservesCRLF はファイルが CRLF + 末尾改行で保存されている場合、
// 上書き後も CRLF + 末尾改行が保持されることを確認する。
// 対応要件: R4.4 (改行コード保持)。fileio.Read → kanban.Sort → fileio.AtomicWrite の
// 三段構成が cmd 層で正しく繋がっていることを構造的に検証する。
func TestSort_PreservesCRLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "kanban.md")
	input := "- [x] A\r\n- [ ] B\r\n- [x] C\r\n"
	writeFile(t, path, input)

	cmd := newRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"sort", path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	got := readFile(t, path)
	want := "- [x] A\r\n- [x] C\r\n- [ ] B\r\n"
	if got != want {
		t.Errorf("CRLF 保持失敗: got %q, want %q", got, want)
	}
}
