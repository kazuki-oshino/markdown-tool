// cmd/mdt の sort サブコマンド統合テスト (tasks.md 7.1)。
//
// 検証範囲 (requirements.md R1.4, R4.3, R4.4, R4.6):
//   - 通常書き込み: ファイルが期待通り上書きされ、改行コード (LF / CRLF) と
//     末尾改行有無が保持されること (R4.3, R4.4)
//   - --dry-run: ファイルが不変であり、stdout が diffview.Render の期待出力と
//     完全一致すること (R4.1, R4.2, R4.6)
//   - no-op (入力 == 出力): ファイル不変かつ exit 0 を返すこと (R1.3)
//   - CRLF E2E: testdata/fileio/crlf_preserve.md.in を t.TempDir() 配下に
//     コピーして実 I/O 経路で実行し、testdata/fileio/crlf_preserve.md.out に
//     完全一致すること (R4.4)
//
// 設計上の意図:
//   - sort_test.go は cobra コマンドのユニットテストとして個別の振る舞いを
//     検証する。本ファイルは fileio.Read → kanban.Sort → fileio.AtomicWrite と
//     diffview.Render を束ねた end-to-end フローを「ファイルシステム経由」で
//     検証する位置付けで配置する (design.md "System Flows / mdt sort 主要フロー")。
//   - testdata は repo-root の testdata/fileio/ にあるため、cmd/mdt パッケージから
//     相対パス "../../testdata/fileio/..." で参照する。embed は親ディレクトリ参照不可
//     のため、コピーは os.ReadFile + os.WriteFile で行う。
//   - テストは cobra.Command.SetArgs / SetOut / SetErr 経由で in-process 実行し、
//     終了コードの classifyExitCode 経由検証も保つ (subprocess 起動を避けて
//     `just test` でそのまま実行可能にする)。
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/kazuki-oshino/markdown-tool/internal/diffview"
)

// crlfFixtureRelDir は repo-root testdata/fileio/ への cmd/mdt からの相対パス。
// 関数名定数として一箇所に集めることで、testdata 移設時の修正点を局所化する。
const crlfFixtureRelDir = "../../testdata/fileio"

// copyFixtureToTempDir は repo-root の testdata/fileio/<name> を t.TempDir() 配下へ
// バイナリ等価でコピーし、コピー先の絶対パスを返す。
//
// 改行コード / 末尾改行有無を 1 byte たりとも改変しないことが本テストの前提なので、
// 必ず os.ReadFile + os.WriteFile (バイナリ I/O) で行う (text 変換層を挟まない)。
func copyFixtureToTempDir(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join(crlfFixtureRelDir, name)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("fixture 読み込み失敗 %q: %v", src, err)
	}
	dst := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("fixture コピー失敗 %q: %v", dst, err)
	}
	return dst
}

// readBytes はバイナリ I/O でファイル内容を読み込む。
// 改行コード保持の検証は string 比較ではなく []byte 等価で行う方が壊れにくい。
func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ファイル読み込み失敗 %q: %v", path, err)
	}
	return b
}

// runSort は newRootCommand を構築して `mdt sort <args...>` を実行し、
// stdout / stderr / err を返す。終了コードは classifyExitCode で分類して呼出側で検証する。
//
// 構造的な利点:
//   - cobra の Execute から得た err を classifyExitCode に通すことで、
//     unit テストと同じ「終了コード分類契約」を統合テストでも維持できる (R5.3)。
func runSort(t *testing.T, args ...string) (stdout, stderr *bytes.Buffer, exitCode int, err error) {
	t.Helper()
	cmd := newRootCommand()
	stdout = &bytes.Buffer{}
	stderr = &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(append([]string{"sort"}, args...))
	err = cmd.Execute()
	exitCode = classifyExitCode(err)
	return stdout, stderr, exitCode, err
}

// TestIntegration_Sort_LF_TrailingEOL は LF + 末尾改行ありのインライン生成入力を
// `mdt sort <file>` で上書き実行し、結果バイトが LF + 末尾改行を保持することを検証する。
// 対応要件: R4.3 (上書き)、R4.4 (改行コード / 末尾改行保持)。
func TestIntegration_Sort_LF_TrailingEOL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lf_eol.md")
	input := []byte("- [x] A\n- [ ] B\n- [x] C\n")
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatalf("入力ファイル生成失敗: %v", err)
	}

	stdout, stderr, code, err := runSort(t, path)
	if err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}
	if stdout.Len() != 0 {
		t.Errorf("通常書き込みでは stdout は空であるべき: %q", stdout.String())
	}

	want := []byte("- [x] A\n- [x] C\n- [ ] B\n")
	if got := readBytes(t, path); !bytes.Equal(got, want) {
		t.Errorf("ファイル内容不一致:\n got=%q\nwant=%q", got, want)
	}
}

// TestIntegration_Sort_LF_NoTrailingEOL は LF + 末尾改行なしのインライン生成入力で
// 末尾改行が「無い」状態が保持されることを検証する。
// 対応要件: R4.4 (末尾改行有無の保持)。
func TestIntegration_Sort_LF_NoTrailingEOL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lf_no_eol.md")
	// 末尾改行なし: 最後の "- [x] C" の後ろに \n を付けない。
	input := []byte("- [x] A\n- [ ] B\n- [x] C")
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatalf("入力ファイル生成失敗: %v", err)
	}

	_, stderr, code, err := runSort(t, path)
	if err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}

	// 末尾改行が無いまま並べ替えだけが行われていることを byte 等価で検証する。
	want := []byte("- [x] A\n- [x] C\n- [ ] B")
	got := readBytes(t, path)
	if !bytes.Equal(got, want) {
		t.Errorf("末尾改行なしが保持されていない:\n got=%q\nwant=%q", got, want)
	}
	// 末尾 byte が '\n' で終わらないことを構造的にも固定する。
	if len(got) > 0 && got[len(got)-1] == '\n' {
		t.Errorf("末尾改行が付与されてしまった (末尾 byte = %q)", got[len(got)-1])
	}
}

// TestIntegration_Sort_CRLF_FromFixture は repo-root testdata/fileio/crlf_preserve.md.in を
// t.TempDir() にコピーして `mdt sort <file>` を実行し、結果バイトが
// testdata/fileio/crlf_preserve.md.out と完全一致することを検証する。
// 対応要件: R4.3 (上書き)、R4.4 (CRLF 保持)。
//
// fileio.Read → kanban.Sort → fileio.AtomicWrite の三段構成 (design.md
// "改行コード責務の分割") が cmd 層で正しく繋がっており、CRLF + 末尾改行ありの
// 実フィクスチャを byte レベルで保持することの構造的保証を pin する。
func TestIntegration_Sort_CRLF_FromFixture(t *testing.T) {
	path := copyFixtureToTempDir(t, "crlf_preserve.md.in")

	_, stderr, code, err := runSort(t, path)
	if err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}

	want := readBytes(t, filepath.Join(crlfFixtureRelDir, "crlf_preserve.md.out"))
	got := readBytes(t, path)
	if !bytes.Equal(got, want) {
		t.Errorf("CRLF fixture E2E 不一致:\n got=%q\nwant=%q", got, want)
	}
}

// TestIntegration_Sort_DryRun_StdoutMatchesDiffview は --dry-run が
// (a) ファイルを不変に保ち、(b) stdout が diffview.Render(content, sorted) と
// 完全一致することを検証する。
// 対応要件: R4.1 (差分プレビュー)、R4.2 (識別可能フォーマット)、R4.6 (ファイル不変)。
//
// 期待 diff は diffview.Render を直接呼んで算出する (ハードコード比較ではなく
// 実装契約そのものを参照点にする)。これにより diffview の出力フォーマットが
// 進化しても本テストは契約レベルで追従する。
func TestIntegration_Sort_DryRun_StdoutMatchesDiffview(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dryrun.md")
	input := []byte("- [x] A\n- [ ] B\n- [x] C\n")
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatalf("入力ファイル生成失敗: %v", err)
	}

	stdout, stderr, code, err := runSort(t, "--dry-run", path)
	if err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}

	// (a) ファイルが不変であること (R4.6)。
	if got := readBytes(t, path); !bytes.Equal(got, input) {
		t.Errorf("--dry-run でファイルが変更された:\n got=%q\nwant=%q", got, input)
	}

	// (b) stdout が diffview.Render の期待出力と完全一致 (R4.1, R4.2)。
	// kanban.Sort の確定的な出力 (LF 統一) を期待値に直接埋め込み、
	// diffview.Render(before, after) の戻り値と byte 等価で比較する。
	wantSorted := "- [x] A\n- [x] C\n- [ ] B\n"
	wantDiff := diffview.Render(string(input), wantSorted)
	if got := stdout.String(); got != wantDiff {
		t.Errorf("stdout が diffview.Render と一致しない:\n got=%q\nwant=%q", got, wantDiff)
	}
}

// TestIntegration_Sort_NoOp_FileUnchanged は入力 == 出力ケース (移動可能な [x] が
// 1 つも存在しない: ここでは [ ] が一切ない入力) でファイル不変・stdout 空・exit 0 を
// 検証する。
// 対応要件: R1.3 (移動対象なしならファイル不変)、R2.2 ([ ] が一切ない場合)。
func TestIntegration_Sort_NoOp_FileUnchanged(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "noop.md")
	// [ ] が一切ない → kanban.Sort は早期リターンで input をそのまま返す (no-op)。
	input := []byte("- [x] A\n- [x] B\n")
	if err := os.WriteFile(path, input, 0o644); err != nil {
		t.Fatalf("入力ファイル生成失敗: %v", err)
	}

	stdout, stderr, code, err := runSort(t, path)
	if err != nil {
		t.Fatalf("Execute: %v (stderr=%q)", err, stderr.String())
	}
	if code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}
	if stdout.Len() != 0 {
		t.Errorf("no-op では stdout は空であるべき: %q", stdout.String())
	}
	if got := readBytes(t, path); !bytes.Equal(got, input) {
		t.Errorf("no-op でファイルが変更された:\n got=%q\nwant=%q", got, input)
	}
}
