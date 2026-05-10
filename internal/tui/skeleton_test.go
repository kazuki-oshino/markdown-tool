// Package tui のスケルトンに対する teatest ベースのヘッドレステスト。
//
// 検証範囲:
//   - 初期画面表示 (R6.3): View() が固定テキストを返し、Bubble Tea の出力ストリームに描画される
//   - q キー押下で終了 (R6.2): Update() が tea.Quit を返してプログラムが完了する
//   - Ctrl+C で終了 (R6.2): Update() が tea.Quit を返してプログラムが完了する
//
// 公開関数 Run() error 自体は TTY を必要とするため直接呼び出さず、
// 同じ初期モデル newModel() を teatest.NewTestModel に渡すことで構成的に検証する。
package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

const (
	// 各テスト共通のタイムアウト。teatest の WaitFor / WaitFinished に渡す。
	testWaitDuration  = 2 * time.Second
	testCheckInterval = 20 * time.Millisecond
	testFinalTimeout  = 2 * time.Second
)

// waitForInitialView は初期画面の固定テキストが Bubble Tea の出力に
// 現れるまで待つヘルパ。3 ケース共通の前提条件。
func waitForInitialView(t *testing.T, tm *teatest.TestModel) {
	t.Helper()
	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(initialView))
	}, teatest.WithDuration(testWaitDuration), teatest.WithCheckInterval(testCheckInterval))
}

// TestSkeletonInitialView は起動直後に固定の初期画面テキストが描画されることを確認する。
// 対応要件: R6.3 (起動・初期画面表示・終了のみのスケルトン)。
func TestSkeletonInitialView(t *testing.T) {
	tm := teatest.NewTestModel(t, newModel(), teatest.WithInitialTermSize(80, 24))

	waitForInitialView(t, tm)

	// このテストはキー入力での終了を検証しないため、後始末として Quit を呼ぶ。
	if err := tm.Quit(); err != nil {
		t.Fatalf("tm.Quit returned error: %v", err)
	}
	tm.WaitFinished(t, teatest.WithFinalTimeout(testFinalTimeout))
}

// TestSkeletonQuitsOnQ は 'q' キー押下で tea.Quit が返り、プログラムが完了することを確認する。
// 対応要件: R6.2 (q キーで明示終了できる手段の提供)。
//
// 検証ロジック: q を送信後 WaitFinished がタイムアウトせず返ること =
// Update() が tea.Quit を返してプログラムループが正常に終了したことを示す。
func TestSkeletonQuitsOnQ(t *testing.T) {
	tm := teatest.NewTestModel(t, newModel(), teatest.WithInitialTermSize(80, 24))

	waitForInitialView(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(testFinalTimeout))
	if fm := tm.FinalModel(t); fm == nil {
		t.Fatal("expected non-nil final model after q press")
	}
}

// TestSkeletonQuitsOnCtrlC は Ctrl+C で tea.Quit が返り、プログラムが完了することを確認する。
// 対応要件: R6.2 (Ctrl+C で明示終了できる手段の提供)。
func TestSkeletonQuitsOnCtrlC(t *testing.T) {
	tm := teatest.NewTestModel(t, newModel(), teatest.WithInitialTermSize(80, 24))

	waitForInitialView(t, tm)

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	tm.WaitFinished(t, teatest.WithFinalTimeout(testFinalTimeout))
	if fm := tm.FinalModel(t); fm == nil {
		t.Fatal("expected non-nil final model after Ctrl+C")
	}
}
