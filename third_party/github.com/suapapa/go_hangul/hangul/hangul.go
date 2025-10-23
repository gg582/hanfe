package hangul

var (
	leadingJamo  = []rune{'ㄱ', 'ㄲ', 'ㄴ', 'ㄷ', 'ㄸ', 'ㄹ', 'ㅁ', 'ㅂ', 'ㅃ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅉ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
	medialJamo   = []rune{'ㅏ', 'ㅐ', 'ㅑ', 'ㅒ', 'ㅓ', 'ㅔ', 'ㅕ', 'ㅖ', 'ㅗ', 'ㅘ', 'ㅙ', 'ㅚ', 'ㅛ', 'ㅜ', 'ㅝ', 'ㅞ', 'ㅟ', 'ㅠ', 'ㅡ', 'ㅢ', 'ㅣ'}
	trailingJamo = []rune{0, 'ㄱ', 'ㄲ', 'ㄳ', 'ㄴ', 'ㄵ', 'ㄶ', 'ㄷ', 'ㄹ', 'ㄺ', 'ㄻ', 'ㄼ', 'ㄽ', 'ㄾ', 'ㄿ', 'ㅀ', 'ㅁ', 'ㅂ', 'ㅄ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}

	leadingIndex  = make(map[rune]int)
	medialIndex   = make(map[rune]int)
	trailingIndex = make(map[rune]int)
)

func init() {
	for i, r := range leadingJamo {
		leadingIndex[r] = i
	}
	for i, r := range medialJamo {
		medialIndex[r] = i
	}
	for i, r := range trailingJamo {
		if r != 0 {
			trailingIndex[r] = i
		}
	}
	trailingIndex[0] = 0
}

const (
	baseCodePoint = 0xAC00
	leadingCount  = 19
	medialCount   = 21
	trailingCount = 28
)

func Compose(leading, medial, trailing rune) (rune, bool) {
	li, ok := leadingIndex[leading]
	if !ok {
		return 0, false
	}
	mi, ok := medialIndex[medial]
	if !ok {
		return 0, false
	}
	ti, ok := trailingIndex[trailing]
	if !ok {
		return 0, false
	}
	composed := baseCodePoint + rune((li*medialCount+mi)*trailingCount+ti)
	return composed, true
}

func IsLeading(r rune) bool {
	_, ok := leadingIndex[r]
	return ok
}

func IsMedial(r rune) bool {
	_, ok := medialIndex[r]
	return ok
}

func IsTrailing(r rune) bool {
	if r == 0 {
		return true
	}
	_, ok := trailingIndex[r]
	return ok
}

func LeadingList() []rune {
	return append([]rune(nil), leadingJamo...)
}

func MedialList() []rune {
	return append([]rune(nil), medialJamo...)
}

func TrailingList() []rune {
	return append([]rune(nil), trailingJamo...)
}
