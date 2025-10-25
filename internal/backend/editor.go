package backend

import "unicode/utf8"

const (
	legacyEditorWidth  = 80
	legacyEditorHeight = 16
	legacyTitleLimit   = 120
	legacyBodyLimit    = 4096

	editorExpansionFactor = 5
)

type EditorKind string

const (
	EditorKindBBS     EditorKind = "bbs"
	EditorKindASCII   EditorKind = "ascii"
	EditorKindProfile EditorKind = "profile"
)

type EditorDimensions struct {
	Width  int
	Height int
}

type EditorLimits struct {
	TitleBytes int
	BodyBytes  int
}

type EditorConfig struct {
	Kind              EditorKind
	Dimensions        EditorDimensions
	Limits            EditorLimits
	SupportsMultiline bool
}

func NewBBSEditorConfig() EditorConfig {
	return EditorConfig{
		Kind: EditorKindBBS,
		Dimensions: EditorDimensions{
			Width:  legacyEditorWidth * editorExpansionFactor,
			Height: legacyEditorHeight * editorExpansionFactor,
		},
		Limits: EditorLimits{
			TitleBytes: legacyTitleLimit * editorExpansionFactor,
			BodyBytes:  legacyBodyLimit * editorExpansionFactor,
		},
		SupportsMultiline: true,
	}
}

func NewASCIIEditorConfig() EditorConfig {
	cfg := NewBBSEditorConfig()
	cfg.Kind = EditorKindASCII
	return cfg
}

func NewProfileEditorConfig() EditorConfig {
	cfg := NewBBSEditorConfig()
	cfg.Kind = EditorKindProfile
	return cfg
}

func (cfg EditorConfig) LegacyCompatible() bool {
	return cfg.Dimensions.Width >= legacyEditorWidth &&
		cfg.Dimensions.Height >= legacyEditorHeight &&
		cfg.Limits.TitleBytes >= legacyTitleLimit &&
		cfg.Limits.BodyBytes >= legacyBodyLimit &&
		cfg.SupportsMultiline
}

func (cfg EditorConfig) CanAccept(title, body string) bool {
	return len(title) <= cfg.Limits.TitleBytes && len(body) <= cfg.Limits.BodyBytes
}

func (cfg EditorConfig) TrimToLimits(title, body string) (string, string) {
	return trimToBytes(title, cfg.Limits.TitleBytes), trimToBytes(body, cfg.Limits.BodyBytes)
}

func trimToBytes(input string, limit int) string {
	if limit <= 0 || len(input) <= limit {
		return input
	}

	buf := make([]byte, 0, limit)
	for len(input) > 0 {
		r, size := utf8.DecodeRuneInString(input)
		if r == utf8.RuneError && size == 1 {
			break
		}
		if len(buf)+size > limit {
			break
		}
		buf = append(buf, input[:size]...)
		input = input[size:]
	}
	return string(buf)
}
