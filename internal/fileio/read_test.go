package fileio

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// writeBytes は t.TempDir() 配下にバイト列をそのまま書き出すヘルパ。
// 改行コードを保ったまま固定したいので os.WriteFile を直接利用する。
func writeBytes(t *testing.T, dir, name string, body []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("テスト用ファイル書き込みに失敗: %v", err)
	}
	return path
}

// TestRead_LineEnding_LF_OnlyTrailing は LF のみ・末尾改行ありの基本形。
// 期待: content は LF 統一文字列のまま、LineEnding=LF、HasTrailingEOL=true。
func TestRead_LineEnding_LF_OnlyTrailing(t *testing.T) {
	dir := t.TempDir()
	body := []byte("a\nb\nc\n")
	path := writeBytes(t, dir, "lf_trailing.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\nc\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingLF {
		t.Errorf("LineEnding = %v, want LineEndingLF", meta.LineEnding)
	}
	if !meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = false, want true")
	}
}

// TestRead_LineEnding_LF_NoTrailing は LF のみ・末尾改行なしのケース。
// 期待: HasTrailingEOL=false で、content も末尾改行なしのまま。
func TestRead_LineEnding_LF_NoTrailing(t *testing.T) {
	dir := t.TempDir()
	body := []byte("a\nb\nc")
	path := writeBytes(t, dir, "lf_notrailing.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\nc"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingLF {
		t.Errorf("LineEnding = %v, want LineEndingLF", meta.LineEnding)
	}
	if meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = true, want false")
	}
}

// TestRead_LineEnding_CRLF_Trailing は CRLF のみ・末尾改行ありのケース。
// 期待: content は LF 統一に正規化、LineEnding=CRLF、HasTrailingEOL=true。
func TestRead_LineEnding_CRLF_Trailing(t *testing.T) {
	dir := t.TempDir()
	body := []byte("a\r\nb\r\nc\r\n")
	path := writeBytes(t, dir, "crlf_trailing.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\nc\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingCRLF {
		t.Errorf("LineEnding = %v, want LineEndingCRLF", meta.LineEnding)
	}
	if !meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = false, want true")
	}
}

// TestRead_LineEnding_CRLF_NoTrailing は CRLF のみ・末尾改行なしのケース。
func TestRead_LineEnding_CRLF_NoTrailing(t *testing.T) {
	dir := t.TempDir()
	body := []byte("a\r\nb\r\nc")
	path := writeBytes(t, dir, "crlf_notrailing.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\nc"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingCRLF {
		t.Errorf("LineEnding = %v, want LineEndingCRLF", meta.LineEnding)
	}
	if meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = true, want false")
	}
}

// TestRead_LineEnding_Mixed_CRLFDominant は LF と CRLF が混在し、
// CRLF が最頻のケース。design.md と tasks.md 4.1 の「混在は最頻採用」に従い、
// LineEnding は LineEndingCRLF を返す（最頻採用ルール）。
func TestRead_LineEnding_Mixed_CRLFDominant(t *testing.T) {
	dir := t.TempDir()
	// CRLF: 2 個 (a\r\n, b\r\n)、LF: 1 個 (c\n)。最頻 = CRLF。
	body := []byte("a\r\nb\r\nc\nd\r\n")
	path := writeBytes(t, dir, "mixed_crlf_dominant.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\nc\nd\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingCRLF {
		t.Errorf("LineEnding = %v, want LineEndingCRLF (最頻採用: CRLF=3, LF=1)", meta.LineEnding)
	}
	if !meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = false, want true")
	}
}

// TestRead_LineEnding_Mixed_LFDominant は LF が最頻のケース。
func TestRead_LineEnding_Mixed_LFDominant(t *testing.T) {
	dir := t.TempDir()
	// LF: 2 個 (a\n, b\n)、CRLF: 1 個 (c\r\n)。最頻 = LF。
	body := []byte("a\nb\nc\r\n")
	path := writeBytes(t, dir, "mixed_lf_dominant.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\nc\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingLF {
		t.Errorf("LineEnding = %v, want LineEndingLF (最頻採用: LF=2, CRLF=1)", meta.LineEnding)
	}
	if !meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = false, want true")
	}
}

// TestRead_LineEnding_Mixed_Tie は CRLF と LF の数が同数の場合。
// design.md / tasks.md は同数時の挙動を明示しないため、決定性のためタイブレークは
// LF とする方針で固定する（実装ノートに残す）。Read のテストではこの規約を契約として
// 固定し、回帰検出に用いる。
func TestRead_LineEnding_Mixed_Tie(t *testing.T) {
	dir := t.TempDir()
	// CRLF: 1 個、LF: 1 個。同数 → LF。
	body := []byte("a\r\nb\n")
	path := writeBytes(t, dir, "mixed_tie.md", body)

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "a\nb\n"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingLF {
		t.Errorf("LineEnding = %v, want LineEndingLF (同数時のタイブレーク)", meta.LineEnding)
	}
}

// TestRead_Empty は空ファイルのケース。改行コード判別不能 → LF をデフォルトとする。
func TestRead_Empty(t *testing.T) {
	dir := t.TempDir()
	path := writeBytes(t, dir, "empty.md", []byte(""))

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if content != "" {
		t.Errorf("content = %q, want \"\"", content)
	}
	if meta.LineEnding != LineEndingLF {
		t.Errorf("LineEnding = %v, want LineEndingLF (空ファイル時のデフォルト)", meta.LineEnding)
	}
	if meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = true, want false (空ファイル)")
	}
}

// TestRead_NoNewline は改行を含まない 1 行ファイルのケース。
// LF/CRLF どちらも 0 件 → LF をデフォルトとし、HasTrailingEOL=false。
func TestRead_NoNewline(t *testing.T) {
	dir := t.TempDir()
	path := writeBytes(t, dir, "single_line.md", []byte("hello"))

	content, meta, err := Read(path)
	if err != nil {
		t.Fatalf("Read が失敗: %v", err)
	}
	if got, want := content, "hello"; got != want {
		t.Errorf("content = %q, want %q", got, want)
	}
	if meta.LineEnding != LineEndingLF {
		t.Errorf("LineEnding = %v, want LineEndingLF (改行なしファイルのデフォルト)", meta.LineEnding)
	}
	if meta.HasTrailingEOL {
		t.Errorf("HasTrailingEOL = true, want false")
	}
}

// TestRead_NotFound は存在しないファイルパスを与えたときに、
// fmt.Errorf("...: %w", err) で wrap されたエラーが返ることを検証する。
// errors.Is(err, fs.ErrNotExist) が true になることを契約として固定する。
func TestRead_NotFound(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "does_not_exist.md")

	content, meta, err := Read(missing)
	if err == nil {
		t.Fatalf("Read が成功してしまった: content=%q meta=%+v", content, meta)
	}
	// %w で wrap されていることを errors.Is 経由で検証する。
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("err = %v, want os.ErrNotExist で wrap されていること", err)
	}
}
