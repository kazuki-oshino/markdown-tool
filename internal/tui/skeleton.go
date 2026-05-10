// Package tui は引数なし起動時の Bubble Tea スケルトンを提供する。
//
// 役割:
//   - 起動・初期画面表示・終了のみを担う最小実装 (R6.1, R6.3)
//   - q キーまたは Ctrl+C で tea.Quit を返し、明示的に終了できる (R6.2)
//   - cmd/mdt ルートの「サブコマンド未指定 & 引数なし」分岐から Run() を呼ばれ、
//     正常終了時は nil を返す (R6.4)
//
// design.md "TUI Adapter / internal/tui" の Service Interface を実装する。
// ファイル選択ブラウザ・編集操作・複数画面遷移などのリッチ UX は本スケルトンの範囲外 (R6.3)。
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// initialView は初期画面に表示する固定テキスト。
//
// design.md の "Responsibilities & Constraints" で例示されている文言を採用する。
// テスト (skeleton_test.go) はこの定数を介して描画内容と終了挙動を検証する。
const initialView = "mdt - press q or Ctrl+C to quit"

// model は Bubble Tea のスケルトンモデル。
//
// 不変条件 (design.md "Invariants"): スケルトンはモデル状態を持たない。
// すなわち Update は終了以外で状態遷移を起こさず、View は常に initialView を返す。
type model struct{}

// newModel は初期モデルを返す。
//
// 公開しない構造体 model のゼロ値を返すラッパで、
// Run() および teatest 経由のテストの双方が同じ初期化経路を使えるようにする。
func newModel() model {
	return model{}
}

// Init はスケルトン起動時の初期コマンドを返す。
// 状態遷移を持たないため nil を返す。
func (m model) Init() tea.Cmd {
	return nil
}

// Update はキー入力に応じてプログラム終了を要求する。
//
// 受理する入力 (R6.2):
//   - tea.KeyMsg.String() == "q":      'q' キー押下
//   - tea.KeyMsg.String() == "ctrl+c": Ctrl+C 押下
//
// 上記以外のメッセージは無視し、状態を変更しない (R6.3 / 編集操作なし)。
// String() ベースで判定することで、tea.KeyRunes / tea.KeyCtrlC の Type 比較を
// 個別に書き分けず、Bubble Tea のキー表現規約に整合させる。
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// View は固定の初期画面テキストを返す (R6.3)。
// モデル状態に依存しないため常に同一の文字列を返す。
func (m model) View() string {
	return initialView
}

// Run は Bubble Tea スケルトンを起動し、利用者が q または Ctrl+C で
// 終了するまでブロックする。
//
// 戻り値:
//   - nil:    正常終了 (利用者の終了操作で tea.Quit が返り、プログラムループが完了した場合)
//   - error:  Bubble Tea ランタイムが返したエラーをそのまま伝搬する
//
// design.md "TUI Adapter" の Postconditions / Invariants に従い、
// 公開 API としては error を返す形に統一する (cmd/mdt 側で終了コードに変換)。
func Run() error {
	if _, err := tea.NewProgram(newModel()).Run(); err != nil {
		return err
	}
	return nil
}
