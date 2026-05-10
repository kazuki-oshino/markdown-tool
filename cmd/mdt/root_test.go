// cmd/mdt のルートコマンドおよび終了コード規約に対するユニットテスト。
//
// 検証範囲:
//   - --help / -h で usage が stdout に表示され、エラーは nil (R5.2)
//   - 引数なし（サブコマンドも引数も指定なし）で TUI 起動分岐が呼ばれる (R6.1, R6.4)
//   - 未知フラグは UsageError 分類のエラーを返す（main 側で exit 2 にマップされる、R5.3）
//
// 設計上のテスタビリティ確保 (design.md "TUI Adapter" / R6.1, R6.4):
//   - tui.Run を直接呼ぶと TTY が必要になりテストが破綻するため、
//     パッケージ変数 runTUI に差し替え可能なフェイクを注入する。
package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// withFakeTUI は runTUI を fake に差し替え、テスト後に復元する。
// 戻り値の関数は呼び出し回数の取得用。
func withFakeTUI(t *testing.T, fakeErr error) (callCount func() int, restore func()) {
	t.Helper()
	original := runTUI
	count := 0
	runTUI = func() error {
		count++
		return fakeErr
	}
	return func() int { return count }, func() { runTUI = original }
}

// TestRootHelpExitsZeroWithUsageOnStdout は --help でエラーなく usage が stdout に出ることを確認する。
// 対応要件: R5.2 (--help / -h で利用方法表示・終了コード 0)。
func TestRootHelpExitsZeroWithUsageOnStdout(t *testing.T) {
	_, restore := withFakeTUI(t, nil)
	defer restore()

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("--help でエラーが返った: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Usage:") {
		t.Errorf("stdout に \"Usage:\" が含まれない: %q", out)
	}
	if !strings.Contains(out, "mdt") {
		t.Errorf("stdout に \"mdt\" が含まれない: %q", out)
	}
}

// TestRootShortHelpFlag は -h でも --help と同等の usage 出力になることを確認する。
// 対応要件: R5.2。
func TestRootShortHelpFlag(t *testing.T) {
	_, restore := withFakeTUI(t, nil)
	defer restore()

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"-h"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("-h でエラーが返った: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("stdout に \"Usage:\" が含まれない: %q", stdout.String())
	}
}

// TestRootNoArgsInvokesTUI は引数なし起動で runTUI が 1 回呼ばれることを確認する。
// 対応要件: R6.1 (引数なし起動で TUI 起動), R6.4 (TUI 終了で exit 0)。
func TestRootNoArgsInvokesTUI(t *testing.T) {
	callCount, restore := withFakeTUI(t, nil)
	defer restore()

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("引数なし実行で予期せぬエラー: %v", err)
	}
	if got := callCount(); got != 1 {
		t.Errorf("runTUI 呼び出し回数 = %d, want 1", got)
	}
}

// TestRootNoArgsTUIErrorPropagates は TUI が error を返した場合、Execute も error を返すことを確認する。
// main 側ではこれが ExitRuntimeError = 1 にマップされる想定 (R6.4 の補足保証)。
func TestRootNoArgsTUIErrorPropagates(t *testing.T) {
	wantErr := errors.New("tui boom")
	_, restore := withFakeTUI(t, wantErr)
	defer restore()

	cmd := newRootCommand()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("TUI error が伝搬しなかった")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want wraps %v", err, wantErr)
	}
}

// TestRootUnknownFlagReturnsUsageError は未知フラグで UsageError 分類のエラーを返すことを確認する。
// 対応要件: R5.3 (使用方法不正は非ゼロ終了。本実装では exit 2 にマップする)。
func TestRootUnknownFlagReturnsUsageError(t *testing.T) {
	_, restore := withFakeTUI(t, nil)
	defer restore()

	cmd := newRootCommand()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs([]string{"--no-such-flag"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("未知フラグでエラーが返らなかった")
	}
	if !isUsageError(err) {
		t.Errorf("err = %v, want classifiable as usage error", err)
	}
}

// TestClassifyExitCode は分類ヘルパが usage / runtime / nil を正しいコードに対応付けることを確認する。
// 対応要件: R5.3 (成功 0 / 使用方法不正 2 / 実行時エラー 1)。
func TestClassifyExitCode(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"nil なら ExitOK", nil, ExitOK},
		{"usage error は ExitUsageError", &usageError{err: errors.New("unknown flag: --x")}, ExitUsageError},
		{"その他は ExitRuntimeError", errors.New("io failure"), ExitRuntimeError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyExitCode(tt.err)
			if got != tt.want {
				t.Errorf("classifyExitCode(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

// TestExitCodeConstants は終了コード定数の値を契約として固定する。
// 対応要件: R5.3。
func TestExitCodeConstants(t *testing.T) {
	if ExitOK != 0 {
		t.Errorf("ExitOK = %d, want 0", ExitOK)
	}
	if ExitRuntimeError != 1 {
		t.Errorf("ExitRuntimeError = %d, want 1", ExitRuntimeError)
	}
	if ExitUsageError != 2 {
		t.Errorf("ExitUsageError = %d, want 2", ExitUsageError)
	}
}
