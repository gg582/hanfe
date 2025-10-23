package types

type InputMode int

const (
	ModeHangul InputMode = iota
	ModeLatin
)

func (m InputMode) String() string {
	switch m {
	case ModeHangul:
		return "hangul"
	case ModeLatin:
		return "latin"
	default:
		return "unknown"
	}
}
