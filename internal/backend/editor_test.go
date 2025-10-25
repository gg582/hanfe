package backend

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestNewBBSEditorConfigScaling(t *testing.T) {
	cfg := NewBBSEditorConfig()

	if cfg.Dimensions.Width != legacyEditorWidth*editorExpansionFactor {
		t.Fatalf("unexpected width: got %d want %d", cfg.Dimensions.Width, legacyEditorWidth*editorExpansionFactor)
	}
	if cfg.Dimensions.Height != legacyEditorHeight*editorExpansionFactor {
		t.Fatalf("unexpected height: got %d want %d", cfg.Dimensions.Height, legacyEditorHeight*editorExpansionFactor)
	}
	if cfg.Limits.TitleBytes != legacyTitleLimit*editorExpansionFactor {
		t.Fatalf("unexpected title limit: got %d want %d", cfg.Limits.TitleBytes, legacyTitleLimit*editorExpansionFactor)
	}
	if cfg.Limits.BodyBytes != legacyBodyLimit*editorExpansionFactor {
		t.Fatalf("unexpected body limit: got %d want %d", cfg.Limits.BodyBytes, legacyBodyLimit*editorExpansionFactor)
	}
	if !cfg.SupportsMultiline {
		t.Fatal("bbs editor must support multiline editing")
	}
	if !cfg.LegacyCompatible() {
		t.Fatal("bbs editor should remain compatible with legacy limits")
	}
}

func TestASCIIAndProfileMirrorBBS(t *testing.T) {
	bbs := NewBBSEditorConfig()
	ascii := NewASCIIEditorConfig()
	profile := NewProfileEditorConfig()

	if ascii.Dimensions != bbs.Dimensions || ascii.Limits != bbs.Limits {
		t.Fatal("ascii editor must share dimensions and limits with the bbs editor")
	}
	if profile.Dimensions != bbs.Dimensions || profile.Limits != bbs.Limits {
		t.Fatal("profile editor must share dimensions and limits with the bbs editor")
	}
	if !ascii.SupportsMultiline || !profile.SupportsMultiline {
		t.Fatal("ascii and profile editors must support multiline editing")
	}
}

func TestTrimToLimitsPreservesRunes(t *testing.T) {
	cfg := NewBBSEditorConfig()

	overTitle := strings.Repeat("가", (cfg.Limits.TitleBytes/3)+10)
	overBody := strings.Repeat("나", (cfg.Limits.BodyBytes/3)+20)

	trimmedTitle, trimmedBody := cfg.TrimToLimits(overTitle, overBody)

	if len(trimmedTitle) > cfg.Limits.TitleBytes {
		t.Fatalf("trimmed title exceeds limit: %d > %d", len(trimmedTitle), cfg.Limits.TitleBytes)
	}
	if len(trimmedBody) > cfg.Limits.BodyBytes {
		t.Fatalf("trimmed body exceeds limit: %d > %d", len(trimmedBody), cfg.Limits.BodyBytes)
	}
	if !utf8.ValidString(trimmedTitle) {
		t.Fatal("trimmed title is not valid utf-8")
	}
	if !utf8.ValidString(trimmedBody) {
		t.Fatal("trimmed body is not valid utf-8")
	}
	if cfg.CanAccept(trimmedTitle, trimmedBody) != true {
		t.Fatal("trimmed content should fit within editor limits")
	}
}
