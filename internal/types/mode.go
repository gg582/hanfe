package types

type InputMode int

const (
	ModeHangul InputMode = iota
	ModeLatin
	ModeKana
	ModeDatabase
)

func (m InputMode) String() string {
	switch m {
	case ModeHangul:
		return "hangul"
	case ModeLatin:
		return "latin"
	case ModeKana:
		return "kana"
	case ModeDatabase:
		return "database"
	default:
		return "unknown"
	}
}
