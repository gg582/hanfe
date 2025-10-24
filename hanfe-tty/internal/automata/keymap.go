package automata

type keyEntry struct {
	lead  rune
	tail  rune
	vowel rune
}

var keyTable = map[rune]keyEntry{
	'q': {lead: 'ㅂ', tail: 'ㅂ'},
	'Q': {lead: 'ㅃ'},
	'w': {lead: 'ㅈ', tail: 'ㅈ'},
	'W': {lead: 'ㅉ'},
	'e': {lead: 'ㄷ', tail: 'ㄷ'},
	'E': {lead: 'ㄸ'},
	'r': {lead: 'ㄱ', tail: 'ㄱ'},
	'R': {lead: 'ㄲ', tail: 'ㄲ'},
	't': {lead: 'ㅅ', tail: 'ㅅ'},
	'T': {lead: 'ㅆ', tail: 'ㅆ'},
	'a': {lead: 'ㅁ', tail: 'ㅁ'},
	's': {lead: 'ㄴ', tail: 'ㄴ'},
	'd': {lead: 'ㅇ', tail: 'ㅇ'},
	'f': {lead: 'ㄹ', tail: 'ㄹ'},
	'g': {lead: 'ㅎ', tail: 'ㅎ'},
	'z': {lead: 'ㅋ', tail: 'ㅋ'},
	'x': {lead: 'ㅌ', tail: 'ㅌ'},
	'c': {lead: 'ㅊ', tail: 'ㅊ'},
	'v': {lead: 'ㅍ', tail: 'ㅍ'},
	'h': {vowel: 'ㅗ'},
	'H': {vowel: 'ㅗ'},
	'j': {vowel: 'ㅓ'},
	'J': {vowel: 'ㅓ'},
	'k': {vowel: 'ㅏ'},
	'K': {vowel: 'ㅏ'},
	'l': {vowel: 'ㅣ'},
	'L': {vowel: 'ㅣ'},
	'y': {vowel: 'ㅛ'},
	'Y': {vowel: 'ㅛ'},
	'u': {vowel: 'ㅕ'},
	'U': {vowel: 'ㅕ'},
	'i': {vowel: 'ㅑ'},
	'I': {vowel: 'ㅑ'},
	'o': {vowel: 'ㅐ'},
	'O': {vowel: 'ㅒ'},
	'p': {vowel: 'ㅔ'},
	'P': {vowel: 'ㅖ'},
	'b': {vowel: 'ㅠ'},
	'B': {vowel: 'ㅠ'},
	'n': {vowel: 'ㅜ'},
	'N': {vowel: 'ㅜ'},
	'm': {vowel: 'ㅡ'},
	'M': {vowel: 'ㅡ'},
}

func lookupKey(r rune) (keyEntry, bool) {
	entry, ok := keyTable[r]
	return entry, ok
}

func isVowelKey(r rune) bool {
	entry, ok := keyTable[r]
	return ok && entry.vowel != 0
}

func isConsonantKey(r rune) bool {
	entry, ok := keyTable[r]
	return ok && entry.lead != 0
}
