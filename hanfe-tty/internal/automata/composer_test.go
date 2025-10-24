package automata

import "testing"

func TestComposerBoJa(t *testing.T) {
	c := NewComposer()

	if commit, preedit := c.Type('q'); commit != "" || preedit != "ㅂ" {
		t.Fatalf("after 'q' expected preedit ㅂ, got commit=%q preedit=%q", commit, preedit)
	}

	if commit, preedit := c.Type('h'); commit != "" || preedit != "보" {
		t.Fatalf("after 'h' expected preedit 보, got commit=%q preedit=%q", commit, preedit)
	}

	if commit, preedit := c.Type('w'); commit != "" || preedit != "봊" {
		t.Fatalf("after 'w' expected preedit 봊, got commit=%q preedit=%q", commit, preedit)
	}

	if commit, preedit := c.Type('k'); commit != "보" || preedit != "자" {
		t.Fatalf("after 'k' expected commit 보 and preedit 자, got commit=%q preedit=%q", commit, preedit)
	}

	if committed := c.Flush(); committed != "자" {
		t.Fatalf("flush should commit 자, got %q", committed)
	}
}

func TestComposerDoubleFinal(t *testing.T) {
	c := NewComposer()
	c.Type('r')
	c.Type('k')

	if commit, preedit := c.Type('q'); commit != "" || preedit != "갑" {
		t.Fatalf("after 'q' expected preedit 갑, got commit=%q preedit=%q", commit, preedit)
	}

	if commit, preedit := c.Type('t'); commit != "" || preedit != "값" {
		t.Fatalf("after 't' expected preedit 값, got commit=%q preedit=%q", commit, preedit)
	}

	if preedit, ok := c.Backspace(); !ok || preedit != "갑" {
		t.Fatalf("backspace should split double final to 갑, got %q (ok=%v)", preedit, ok)
	}

	if preedit, ok := c.Backspace(); !ok || preedit != "가" {
		t.Fatalf("second backspace should remove final to 가, got %q (ok=%v)", preedit, ok)
	}
}

func TestComposerInitialVowel(t *testing.T) {
	c := NewComposer()

	if commit, preedit := c.Type('k'); commit != "" || preedit != "ㅏ" {
		t.Fatalf("initial vowel should stay as jamo, got commit=%q preedit=%q", commit, preedit)
	}

	if commit, preedit := c.Type('k'); commit != "ㅏ" || preedit != "ㅏ" {
		t.Fatalf("second vowel should commit previous, got commit=%q preedit=%q", commit, preedit)
	}
}

func TestComposerCarryTailToNextSyllable(t *testing.T) {
	c := NewComposer()

	c.Type('r') // ㄱ
	c.Type('k') // ㅏ
	c.Type('t') // ㅅ -> 각

	if commit, preedit := c.Type('k'); commit != "가" || preedit != "사" {
		t.Fatalf("vowel after final should move consonant, got commit=%q preedit=%q", commit, preedit)
	}
}
