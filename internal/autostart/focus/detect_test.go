package focus

import (
	"strings"
	"testing"
)

func TestIsTerminalName(t *testing.T) {
	cases := map[string]bool{
		"Alacritty":                true,
		"kitty":                    true,
		"gnome-terminal":           true,
		"com.raggesilver.BlackBox": true,
		"text-editor":              false,
		"firefox":                  false,
		"terminal-window":          true,
		"Cool-Retro-Term":          true,
		"shell":                    false,
	}
	for input, want := range cases {
		if got := isTerminalName(input); got != want {
			t.Fatalf("isTerminalName(%q)=%v want %v", input, got, want)
		}
	}
}

func TestMaybeTerminalTitle(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"My Terminal", true},
		{"bash", false},
		{"TTY1", true},
		{"Console - htop", true},
		{"Notes", false},
	}
	for _, tc := range cases {
		if got := maybeTerminalTitle(strings.ToLower(tc.in)); got != tc.want {
			t.Fatalf("maybeTerminalTitle(%q)=%v want %v", tc.in, got, tc.want)
		}
	}
}
