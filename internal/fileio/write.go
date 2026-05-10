package fileio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AtomicWrite は LF 統一の content に Meta を再適用し、
// 同一ディレクトリの一時ファイル経由で path をアトミックに上書きする。
//
// 仕様 (tasks.md 4.2 / design.md "Side Effects / internal/fileio" の Service Interface /
//
//	requirements.md R4.3, R4.4, R4.5):
//
//   - 改行コード復元: content の各 \n を Meta.LineEnding に応じた改行コードに復元する。
//     LineEndingLF はそのまま、LineEndingCRLF は \n → \r\n に置換、LineEndingMixed は
//     「Read 経由では発生しない予約値」として LF にフォールバック (tasks.md Implementation
//     Notes task 4.1 参照)。
//   - 末尾改行制御: HasTrailingEOL=true なら復元後の文字列が末尾改行 (LF または CRLF) で
//     終端することを保証し、=false なら末尾の改行 (LF/CRLF いずれか) を 1 回除去する。
//   - アトミック置換: 同一ディレクトリに os.CreateTemp で一時ファイルを作り、Write→Sync→
//     Close 後に os.Rename で path を完全置換する (中間破損禁止 / R4.5)。
//   - 失敗時の不変性: いずれかの段階で失敗したら一時ファイルを os.Remove で除去し、
//     path の元内容を不変のまま保つ (R4.5)。エラーは fmt.Errorf("fileio: AtomicWrite %q: %w", ...)
//     で wrap し、上位 cmd/mdt 層で errors.Is(err, fs.ErrNotExist) 等を判定可能にする。
//   - モード保持: 既存ファイルがある場合は os.Stat でモードを取得し、一時ファイルを
//     os.Chmod で同一モードに揃えてから rename する。Stat 失敗 (新規ファイル時等) は
//     一時ファイル既定モード (0o600) のまま続行する。
//
// content が CRLF を含む場合の挙動は未定義 (呼出側違反: design.md "Preconditions")。
// kanban.Sort の出力規約により LF 統一が保証される前提で動作する。
func AtomicWrite(path string, content string, meta Meta) error {
	body := encodeBody(content, meta)

	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// 同一ディレクトリに一時ファイルを作る。これにより最終 Rename が
	// 同一ファイルシステム上で原子的に実行される (異 FS 間の Rename は EXDEV になる)。
	tmp, err := os.CreateTemp(dir, base+".tmp-*")
	if err != nil {
		return fmt.Errorf("fileio: AtomicWrite %q: %w", path, err)
	}
	tmpPath := tmp.Name()

	// 失敗時の片付けフック。成功時はクリアする (success=true)。
	success := false
	defer func() {
		if !success {
			// Close は冪等に扱う (Write 中に失敗した場合 tmp は既に閉じられている可能性)。
			_ = tmp.Close()
			_ = os.Remove(tmpPath)
		}
	}()

	if _, werr := tmp.Write(body); werr != nil {
		return fmt.Errorf("fileio: AtomicWrite %q: %w", path, werr)
	}
	// Sync で OS バッファを flush しておく。アトミック置換前にディスクへ反映させ、
	// 電源断などの障害時にも一貫性を保つ。
	if serr := tmp.Sync(); serr != nil {
		return fmt.Errorf("fileio: AtomicWrite %q: %w", path, serr)
	}
	if cerr := tmp.Close(); cerr != nil {
		return fmt.Errorf("fileio: AtomicWrite %q: %w", path, cerr)
	}

	// 既存ファイルのモードを保持する (R4.3 上書き挙動の自然な拡張)。
	// Stat 失敗 (path が存在しない / 新規作成) は既定モードのまま続行する。
	if info, statErr := os.Stat(path); statErr == nil {
		if chmodErr := os.Chmod(tmpPath, info.Mode().Perm()); chmodErr != nil {
			return fmt.Errorf("fileio: AtomicWrite %q: %w", path, chmodErr)
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		// 既存ファイルがあるはずなのに Stat が ErrNotExist 以外で失敗するケースは
		// アクセス権限など下位 I/O 異常の可能性があるため、ここで中断する。
		return fmt.Errorf("fileio: AtomicWrite %q: %w", path, statErr)
	}

	if rerr := os.Rename(tmpPath, path); rerr != nil {
		return fmt.Errorf("fileio: AtomicWrite %q: %w", path, rerr)
	}

	success = true
	return nil
}

// encodeBody は LF 統一の content と Meta から、ディスクへ書き出すバイト列を生成する。
//
// 順序:
//  1. HasTrailingEOL=false のとき末尾の \n を 1 回だけ除去する (LF 段階で行うことで
//     CRLF 復元時に \r\n 末尾を取りこぼさない)。
//  2. LineEnding に従い改行コードを復元する (LineEndingMixed は LF フォールバック)。
//
// この順序で実装する理由: CRLF 段階で末尾改行を除去すると、復元前に長さ 2 の改行を
// 切り出す処理が入り煩雑になる。LF 統一段階で末尾を整えてから一括変換する方が、
// 「LF 1 文字を切るだけ」「\n → \r\n を全置換するだけ」という単純な 2 ステップに分解できる。
func encodeBody(content string, meta Meta) []byte {
	s := content
	if !meta.HasTrailingEOL && strings.HasSuffix(s, "\n") {
		s = s[:len(s)-1]
	}

	switch meta.LineEnding {
	case LineEndingCRLF:
		// LF → CRLF 復元。content は LF 統一前提なので素直な全置換でよい。
		s = strings.ReplaceAll(s, "\n", "\r\n")
	case LineEndingLF, LineEndingMixed:
		// LineEndingMixed は Read からは emit されない予約値 (meta.go コメント参照)。
		// 呼出側が明示的に渡したケースでは LF にフォールバックして書き戻す。
		// (panic は cmd/mdt 層から扱いづらいため避ける方針: tasks.md Implementation Notes 4.1)
	}

	return []byte(s)
}
