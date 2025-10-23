package ime

import (
	"testing"

	"github.com/suapapa/go_hangul/keyboard"
)

func TestComposeAnnyeonghaseyo(t *testing.T) {
	composer := NewComposer(keyboard.Dubeolsik())
	input := "dkssudgktpdy"
	for _, r := range input {
		if !composer.TypeKey(r) {
			t.Fatalf("unexpected literal for %c", r)
		}
	}
	got := composer.FlushText()
	want := "안녕하세요"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestBackspaceClearsSyllable(t *testing.T) {
	composer := NewComposer(keyboard.Dubeolsik())
	composer.TypeKey('d') // ㅇ
	composer.TypeKey('k') // ㅏ -> 아
	if got := composer.Text(); got != "아" {
		t.Fatalf("unexpected composed text: %q", got)
	}

	composer.Backspace() // remove ㅏ
	if got := composer.Text(); got != "ㅇ" {
		t.Fatalf("expected lead jamo after removing vowel, got %q", got)
	}

	composer.Backspace() // remove ㅇ
	if got := composer.Text(); got != "" {
		t.Fatalf("expected empty buffer after removing lead, got %q", got)
	}
}

func TestSpaceAndLiteral(t *testing.T) {
	composer := NewComposer(keyboard.Dubeolsik())
	for _, r := range "dkssudgktpdy" {
		composer.TypeKey(r)
	}
	composer.Space()
	composer.AppendLiteral('1')
	got := composer.Text()
	if got != "안녕하세요 1" {
		t.Fatalf("unexpected text: %q", got)
	}
}
