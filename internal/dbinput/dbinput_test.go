package dbinput

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionLookupAndFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pinyin.tsv")
	if err := os.WriteFile(path, []byte("ni\t你\nhao\t好\n"), 0o644); err != nil {
		t.Fatalf("write dict: %v", err)
	}

	dict, err := LoadDictionary(path)
	if err != nil {
		t.Fatalf("load dict: %v", err)
	}

	session := dict.NewSession()
	if pre := session.Append("n"); pre != "n" {
		t.Fatalf("expected preedit n, got %q", pre)
	}
	if pre := session.Append("i"); pre != "ni" {
		t.Fatalf("expected preedit ni, got %q", pre)
	}
	if commit := session.Commit(); commit != "你" {
		t.Fatalf("expected commit 你, got %q", commit)
	}

	if pre := session.Append("x"); pre != "x" {
		t.Fatalf("expected preedit x, got %q", pre)
	}
	if commit := session.Commit(); commit != "x" {
		t.Fatalf("expected fallback commit x, got %q", commit)
	}
}

func TestBackspace(t *testing.T) {
	dict := &Dictionary{}
	session := dict.NewSession()
	session.Append("hao")
	if pre, ok := session.Backspace(); !ok || pre != "ha" {
		t.Fatalf("expected ha after backspace, got %q ok=%v", pre, ok)
	}
	session.Commit()
	if pre := session.Preedit(); pre != "" {
		t.Fatalf("expected cleared preedit, got %q", pre)
	}
}
