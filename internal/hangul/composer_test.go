package hangul

import "testing"

func TestHangulComposerComposeSyllable(t *testing.T) {
	composer := NewHangulComposer()

	result := composer.Feed('ㅎ', RoleAuto)
	if result.Commit != "" {
		t.Fatalf("expected no commit after initial consonant, got %q", result.Commit)
	}
	if result.Preedit != "ㅎ" {
		t.Fatalf("expected preedit 'ㅎ', got %q", result.Preedit)
	}

	result = composer.Feed('ㅏ', RoleAuto)
	if result.Commit != "" {
		t.Fatalf("expected no commit after vowel, got %q", result.Commit)
	}
	if result.Preedit != "하" {
		t.Fatalf("expected preedit '하', got %q", result.Preedit)
	}

	result = composer.Feed('ㄴ', RoleAuto)
	if result.Commit != "" {
		t.Fatalf("expected no commit after trailing consonant, got %q", result.Commit)
	}
	if result.Preedit != "한" {
		t.Fatalf("expected preedit '한', got %q", result.Preedit)
	}

	committed := composer.Flush()
	if committed != "한" {
		t.Fatalf("expected flush to commit '한', got %q", committed)
	}

	if composer.Flush() != "" {
		t.Fatalf("expected subsequent flush to commit nothing")
	}
}

func TestHangulComposerDoubleInitial(t *testing.T) {
	composer := NewHangulComposer()

	composer.Feed('ㄱ', RoleAuto)
	result := composer.Feed('ㄱ', RoleAuto)
	if result.Preedit != "ㄲ" {
		t.Fatalf("expected double initial to form 'ㄲ', got %q", result.Preedit)
	}

	result = composer.Feed('ㅏ', RoleAuto)
	if result.Preedit != "까" {
		t.Fatalf("expected syllable '까', got %q", result.Preedit)
	}
}

func TestHangulComposerDoubleFinal(t *testing.T) {
	composer := NewHangulComposer()

	composer.Feed('ㄱ', RoleAuto)
	composer.Feed('ㅏ', RoleAuto)

	result := composer.Feed('ㅂ', RoleAuto)
	if result.Preedit != "갑" {
		t.Fatalf("expected trailing consonant to produce '갑', got %q", result.Preedit)
	}

	result = composer.Feed('ㅅ', RoleAuto)
	if result.Preedit != "값" {
		t.Fatalf("expected double final to produce '값', got %q", result.Preedit)
	}

	if composer.Flush() != "값" {
		t.Fatalf("expected flush to commit '값'")
	}
}

func TestHangulComposerBackspace(t *testing.T) {
	composer := NewHangulComposer()
	composer.Feed('ㄱ', RoleAuto)
	composer.Feed('ㅏ', RoleAuto)
	composer.Feed('ㅂ', RoleAuto)
	composer.Feed('ㅅ', RoleAuto)

	if preedit, ok := composer.Backspace(); !ok || preedit != "갑" {
		t.Fatalf("expected backspace to split double final to '갑', got %q (ok=%v)", preedit, ok)
	}

	if preedit, ok := composer.Backspace(); !ok || preedit != "가" {
		t.Fatalf("expected backspace to remove trailing consonant to '가', got %q (ok=%v)", preedit, ok)
	}

	if preedit, ok := composer.Backspace(); !ok || preedit != "ㄱ" {
		t.Fatalf("expected backspace to remove vowel leaving 'ㄱ', got %q (ok=%v)", preedit, ok)
	}

	if preedit, ok := composer.Backspace(); !ok || preedit != "" {
		t.Fatalf("expected backspace to clear leading consonant, got %q (ok=%v)", preedit, ok)
	}

	if _, ok := composer.Backspace(); ok {
		t.Fatalf("expected no further backspace edits once empty")
	}
}

func TestHangulComposerForcedRoles(t *testing.T) {
	composer := NewHangulComposer()

	composer.Feed('ㄱ', RoleAuto)
	composer.Feed('ㅏ', RoleAuto)

	result := composer.Feed('ㄱ', RoleTrailing)
	if result.Preedit != "각" {
		t.Fatalf("expected trailing role to attach consonant, got %q", result.Preedit)
	}

	result = composer.Feed('ㄴ', RoleLeading)
	if result.Commit != "각" {
		t.Fatalf("expected commit of previous syllable before forced leading, got %q", result.Commit)
	}
	if result.Preedit != "ㄴ" {
		t.Fatalf("expected new preedit for leading consonant, got %q", result.Preedit)
	}
}

func TestHangulComposerDoubleMedial(t *testing.T) {
	composer := NewHangulComposer()

	composer.Feed('ㅗ', RoleAuto)
	result := composer.Feed('ㅏ', RoleAuto)

	if result.Commit != "" {
		t.Fatalf("expected no commit while composing double medial, got %q", result.Commit)
	}
	if result.Preedit != "와" {
		t.Fatalf("expected composed vowel to yield '와', got %q", result.Preedit)
	}
}
