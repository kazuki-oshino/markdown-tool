package fileio

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// readBytes は AtomicWrite 後の path のバイト列を読み込むヘルパ。
// テストでは「LF/CRLF が想定通りの位置にあるか」「末尾改行有無」など
// バイト単位での厳密な比較を行うため、Read (LF 統一) ではなく os.ReadFile を直接呼ぶ。
func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("テスト検証用 read に失敗: %v", err)
	}
	return body
}

// TestAtomicWrite_LF_Trailing は LF 統一・末尾改行ありの基本ケース。
// design.md "改行コード責務の分割": AtomicWrite は LF 統一 content + Meta から
// 元の改行コードを復元する責務を持つ。LF + 末尾あり ならそのまま書き出す。
func TestAtomicWrite_LF_Trailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lf_trailing.md")
	content := "a\nb\nc\n"

	err := AtomicWrite(path, content, Meta{LineEnding: LineEndingLF, HasTrailingEOL: true})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "a\nb\nc\n"; got != want {
		t.Errorf("disk bytes = %q, want %q", got, want)
	}
}

// TestAtomicWrite_CRLF_Trailing は CRLF 復元・末尾改行ありのケース。
// requirements.md R4.4: 改行コード保持。LF 統一 content の各 \n を \r\n に復元する。
func TestAtomicWrite_CRLF_Trailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crlf_trailing.md")
	content := "a\nb\nc\n"

	err := AtomicWrite(path, content, Meta{LineEnding: LineEndingCRLF, HasTrailingEOL: true})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "a\r\nb\r\nc\r\n"; got != want {
		t.Errorf("disk bytes = %q, want %q", got, want)
	}
}

// TestAtomicWrite_LF_NoTrailing は LF・末尾改行なしのケース。
// 入力 content が末尾 \n を含んでいても HasTrailingEOL=false なら除去する契約。
// requirements.md R4.4: 末尾改行有無の保持。
func TestAtomicWrite_LF_NoTrailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lf_notrailing.md")
	content := "a\nb\n"

	err := AtomicWrite(path, content, Meta{LineEnding: LineEndingLF, HasTrailingEOL: false})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "a\nb"; got != want {
		t.Errorf("disk bytes = %q, want %q", got, want)
	}
}

// TestAtomicWrite_LF_PreserveTrailing は LF・末尾改行ありで content がそのまま流れるケース。
func TestAtomicWrite_LF_PreserveTrailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "lf_keep.md")
	content := "a\nb\n"

	err := AtomicWrite(path, content, Meta{LineEnding: LineEndingLF, HasTrailingEOL: true})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "a\nb\n"; got != want {
		t.Errorf("disk bytes = %q, want %q", got, want)
	}
}

// TestAtomicWrite_CRLF_NoTrailing は CRLF 復元 + 末尾改行除去の合成ケース。
// 入力 "a\nb\n" を「LF→CRLF 復元」してから末尾の CRLF を除去 → "a\r\nb" となる契約。
func TestAtomicWrite_CRLF_NoTrailing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "crlf_notrailing.md")
	content := "a\nb\n"

	err := AtomicWrite(path, content, Meta{LineEnding: LineEndingCRLF, HasTrailingEOL: false})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "a\r\nb"; got != want {
		t.Errorf("disk bytes = %q, want %q", got, want)
	}
}

// TestAtomicWrite_Empty は空 content・末尾改行なしで 0 バイトファイルが生成されることを検証する。
func TestAtomicWrite_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.md")

	err := AtomicWrite(path, "", Meta{LineEnding: LineEndingLF, HasTrailingEOL: false})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	body := readBytes(t, path)
	if len(body) != 0 {
		t.Errorf("disk bytes len = %d, want 0 (空ファイル)", len(body))
	}
}

// TestAtomicWrite_OverwriteExisting は既存ファイルの完全置換を検証する。
// 事前に異なるバイト列を置き、AtomicWrite 後に content + Meta による
// 期待バイト列で完全に上書きされることを確認する (R4.3 上書きはバックアップなし)。
func TestAtomicWrite_OverwriteExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.md")
	if err := os.WriteFile(path, []byte("OLD CONTENT WITH NO NEWLINE"), 0o644); err != nil {
		t.Fatalf("事前ファイル準備に失敗: %v", err)
	}

	err := AtomicWrite(path, "x\ny\n", Meta{LineEnding: LineEndingCRLF, HasTrailingEOL: true})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "x\r\ny\r\n"; got != want {
		t.Errorf("disk bytes = %q, want %q (完全上書きされるべき)", got, want)
	}
}

// TestAtomicWrite_LineEndingMixed_FallbackToLF は呼び出し側が LineEndingMixed を
// 明示的に渡したケースの契約を固定する。tasks.md "Implementation Notes" task 4.1 に
// 従い、Read 経由では発生しない予約値として LF にフォールバックして書き戻す。
// (panic は呼出側にとって扱いづらいため避ける方針。)
func TestAtomicWrite_LineEndingMixed_FallbackToLF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.md")

	err := AtomicWrite(path, "a\nb\n", Meta{LineEnding: LineEndingMixed, HasTrailingEOL: true})
	if err != nil {
		t.Fatalf("AtomicWrite が失敗: %v", err)
	}
	if got, want := string(readBytes(t, path)), "a\nb\n"; got != want {
		t.Errorf("disk bytes = %q, want %q (Mixed は LF フォールバック)", got, want)
	}
}

// TestAtomicWrite_FailureKeepsOriginal は書き込み失敗時の R4.5 契約を検証する。
// 書き込み権限のないディレクトリ (chmod 0o555) を作り、配下に既存ファイルを置いた状態で
// AtomicWrite を呼ぶと:
//   - non-nil error が返る
//   - 元ファイルのバイト列が不変
//   - 同ディレクトリに一時ファイル (.tmp パターン) が残らない
//
// 注意: macOS で root 実行時は os.Chmod の権限制御が効かないことがあるため
// root はスキップする。Windows も os.Chmod の挙動が異なるため対象外とする。
func TestAtomicWrite_FailureKeepsOriginal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows は os.Chmod の権限制御挙動が異なるためスキップ")
	}
	if os.Geteuid() == 0 {
		t.Skip("root 実行時は chmod 0o555 でも書き込めるためスキップ")
	}

	parent := t.TempDir()
	roDir := filepath.Join(parent, "readonly")
	if err := os.Mkdir(roDir, 0o755); err != nil {
		t.Fatalf("readonly 用ディレクトリ作成に失敗: %v", err)
	}
	target := filepath.Join(roDir, "target.md")
	original := []byte("ORIGINAL\nDO_NOT_TOUCH\n")
	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatalf("元ファイル準備に失敗: %v", err)
	}

	// ディレクトリを read-only に落とす (一時ファイルの CreateTemp が失敗するはず)。
	if err := os.Chmod(roDir, 0o555); err != nil {
		t.Fatalf("chmod に失敗: %v", err)
	}
	t.Cleanup(func() {
		// テスト後始末で TempDir が削除できるよう書き込み権限を戻す。
		_ = os.Chmod(roDir, 0o755)
	})

	err := AtomicWrite(target, "NEW\nCONTENT\n", Meta{LineEnding: LineEndingLF, HasTrailingEOL: true})
	if err == nil {
		t.Fatalf("AtomicWrite が成功してしまった: 失敗を期待")
	}

	// (b) 元ファイルが不変であること。
	got, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("元ファイルの読み戻しに失敗: %v", readErr)
	}
	if string(got) != string(original) {
		t.Errorf("元ファイルが破壊された: got=%q want=%q", got, original)
	}

	// (c) 一時ファイルが残っていないこと。本実装は CreateTemp に
	// pattern = filepath.Base(path) + ".tmp-*" を用いる前提で、".tmp-" を含む
	// エントリが残っていれば回収漏れと判定する。
	entries, listErr := os.ReadDir(roDir)
	if listErr != nil {
		t.Fatalf("readonly dir の ReadDir に失敗: %v", listErr)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("一時ファイルが残存している: %s", e.Name())
		}
	}
}

// TestAtomicWrite_FailureMissingDir は書き込み先親ディレクトリが存在しないときに
// non-nil error を返し、wrap された errors.Is(err, fs.ErrNotExist) が成立することを確認する。
// (read.go の "fileio: read %q: %w" と整合する命名規約: "fileio: AtomicWrite %q: %w".)
func TestAtomicWrite_FailureMissingDir(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "no", "such", "dir", "out.md")

	err := AtomicWrite(target, "x\n", Meta{LineEnding: LineEndingLF, HasTrailingEOL: true})
	if err == nil {
		t.Fatalf("AtomicWrite が成功してしまった: 失敗を期待")
	}
	if !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("err = %v, want fs.ErrNotExist で wrap されていること", err)
	}
}

// TestAtomicWrite_RoundTrip_CRLFFixture は testdata/fileio/crlf_preserve.md.in を
// Read → AtomicWrite → 再 Read のラウンドトリップで content / Meta が一致することを検証する。
// design.md "改行コード責務の分割" の三段構成 (Read → kanban.Sort → AtomicWrite) のうち
// Read ↔ AtomicWrite ペアが Meta を完全保持することを契約として固定する。
func TestAtomicWrite_RoundTrip_CRLFFixture(t *testing.T) {
	// このテストは fileio パッケージの testdata ではなく、repo-root の
	// testdata/fileio/ を参照する (tasks.md 1.3 / 4.1 のレイアウト)。
	// 相対パスはテスト実行時 CWD = パッケージディレクトリ前提で 4 階層上を辿る。
	src := filepath.Join("..", "..", "testdata", "fileio", "crlf_preserve.md.in")
	srcBytes, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("testdata 読み込みに失敗: %v", err)
	}

	// 一旦 TempDir にコピーしてから Read → AtomicWrite (上書き) → 再 Read。
	dir := t.TempDir()
	work := filepath.Join(dir, "roundtrip.md")
	if err := os.WriteFile(work, srcBytes, 0o644); err != nil {
		t.Fatalf("作業ファイル準備に失敗: %v", err)
	}

	content1, meta1, err := Read(work)
	if err != nil {
		t.Fatalf("初回 Read 失敗: %v", err)
	}

	if err := AtomicWrite(work, content1, meta1); err != nil {
		t.Fatalf("AtomicWrite 失敗: %v", err)
	}

	// バイトレベルでも元 fixture と一致するはず (CRLF がそのまま復元される)。
	if got := readBytes(t, work); string(got) != string(srcBytes) {
		t.Errorf("ラウンドトリップでバイト列が変化: got=%q want=%q", got, srcBytes)
	}

	content2, meta2, err := Read(work)
	if err != nil {
		t.Fatalf("再 Read 失敗: %v", err)
	}
	if content1 != content2 {
		t.Errorf("content がラウンドトリップで変化: %q -> %q", content1, content2)
	}
	if meta1 != meta2 {
		t.Errorf("Meta がラウンドトリップで変化: %+v -> %+v", meta1, meta2)
	}
}
