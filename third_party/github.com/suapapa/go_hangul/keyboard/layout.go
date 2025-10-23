package keyboard

import "github.com/suapapa/go_hangul/hangul"

type Layout struct {
	Name            string
	ConsonantKeys   map[rune]rune
	VowelKeys       map[rune]rune
	CombineVowels   map[string]rune
	DecomposeVowels map[rune][2]rune
	CombineFinals   map[string]rune
	DecomposeFinals map[rune][2]rune
	SilentLeading   rune
}

func keyPair(a, b rune) string {
	return string([]rune{a, b})
}

func Dubeolsik() Layout {
	consonants := map[rune]rune{
		'r': 'ㄱ',
		'R': 'ㄲ',
		's': 'ㄴ',
		'e': 'ㄷ',
		'E': 'ㄸ',
		'f': 'ㄹ',
		'a': 'ㅁ',
		'q': 'ㅂ',
		'Q': 'ㅃ',
		't': 'ㅅ',
		'T': 'ㅆ',
		'd': 'ㅇ',
		'w': 'ㅈ',
		'W': 'ㅉ',
		'c': 'ㅊ',
		'z': 'ㅋ',
		'x': 'ㅌ',
		'v': 'ㅍ',
		'g': 'ㅎ',
	}

	vowels := map[rune]rune{
		'k': 'ㅏ',
		'o': 'ㅐ',
		'i': 'ㅑ',
		'O': 'ㅒ',
		'j': 'ㅓ',
		'p': 'ㅔ',
		'u': 'ㅕ',
		'P': 'ㅖ',
		'h': 'ㅗ',
		'y': 'ㅛ',
		'n': 'ㅜ',
		'b': 'ㅠ',
		'm': 'ㅡ',
		'l': 'ㅣ',
	}

	combineVowels := map[string]rune{
		keyPair('ㅗ', 'ㅏ'): 'ㅘ',
		keyPair('ㅗ', 'ㅐ'): 'ㅙ',
		keyPair('ㅗ', 'ㅣ'): 'ㅚ',
		keyPair('ㅘ', 'ㅣ'): 'ㅙ',
		keyPair('ㅜ', 'ㅓ'): 'ㅝ',
		keyPair('ㅜ', 'ㅔ'): 'ㅞ',
		keyPair('ㅜ', 'ㅣ'): 'ㅟ',
		keyPair('ㅝ', 'ㅣ'): 'ㅞ',
		keyPair('ㅡ', 'ㅣ'): 'ㅢ',
	}

	decomposeVowels := map[rune][2]rune{
		'ㅘ': {'ㅗ', 'ㅏ'},
		'ㅙ': {'ㅗ', 'ㅐ'},
		'ㅚ': {'ㅗ', 'ㅣ'},
		'ㅝ': {'ㅜ', 'ㅓ'},
		'ㅞ': {'ㅜ', 'ㅔ'},
		'ㅟ': {'ㅜ', 'ㅣ'},
		'ㅢ': {'ㅡ', 'ㅣ'},
	}

	combineFinals := map[string]rune{
		keyPair('ㄱ', 'ㅅ'): 'ㄳ',
		keyPair('ㄴ', 'ㅈ'): 'ㄵ',
		keyPair('ㄴ', 'ㅎ'): 'ㄶ',
		keyPair('ㄹ', 'ㄱ'): 'ㄺ',
		keyPair('ㄹ', 'ㅁ'): 'ㄻ',
		keyPair('ㄹ', 'ㅂ'): 'ㄼ',
		keyPair('ㄹ', 'ㅅ'): 'ㄽ',
		keyPair('ㄹ', 'ㅌ'): 'ㄾ',
		keyPair('ㄹ', 'ㅍ'): 'ㄿ',
		keyPair('ㄹ', 'ㅎ'): 'ㅀ',
		keyPair('ㅂ', 'ㅅ'): 'ㅄ',
	}

	decomposeFinals := map[rune][2]rune{
		'ㄳ': {'ㄱ', 'ㅅ'},
		'ㄵ': {'ㄴ', 'ㅈ'},
		'ㄶ': {'ㄴ', 'ㅎ'},
		'ㄺ': {'ㄹ', 'ㄱ'},
		'ㄻ': {'ㄹ', 'ㅁ'},
		'ㄼ': {'ㄹ', 'ㅂ'},
		'ㄽ': {'ㄹ', 'ㅅ'},
		'ㄾ': {'ㄹ', 'ㅌ'},
		'ㄿ': {'ㄹ', 'ㅍ'},
		'ㅀ': {'ㄹ', 'ㅎ'},
		'ㅄ': {'ㅂ', 'ㅅ'},
	}

	return Layout{
		Name:            "dubeolsik",
		ConsonantKeys:   consonants,
		VowelKeys:       vowels,
		CombineVowels:   combineVowels,
		DecomposeVowels: decomposeVowels,
		CombineFinals:   combineFinals,
		DecomposeFinals: decomposeFinals,
		SilentLeading:   'ㅇ',
	}
}

func Available() []Layout {
	return []Layout{Dubeolsik()}
}

func ByName(name string) (Layout, bool) {
	for _, layout := range Available() {
		if layout.Name == name {
			return layout, true
		}
	}
	return Layout{}, false
}

func (l Layout) ConsonantForKey(key rune) (rune, bool) {
	v, ok := l.ConsonantKeys[key]
	return v, ok
}

func (l Layout) VowelForKey(key rune) (rune, bool) {
	v, ok := l.VowelKeys[key]
	return v, ok
}

func (l Layout) CombineMedial(a, b rune) (rune, bool) {
	v, ok := l.CombineVowels[keyPair(a, b)]
	return v, ok
}

func (l Layout) DecomposeMedial(v rune) (rune, rune, bool) {
	parts, ok := l.DecomposeVowels[v]
	if !ok {
		return 0, 0, false
	}
	return parts[0], parts[1], true
}

func (l Layout) CombineFinal(a, b rune) (rune, bool) {
	v, ok := l.CombineFinals[keyPair(a, b)]
	return v, ok
}

func (l Layout) DecomposeFinal(v rune) (rune, rune, bool) {
	parts, ok := l.DecomposeFinals[v]
	if !ok {
		return 0, 0, false
	}
	return parts[0], parts[1], true
}

func (l Layout) IsLeading(r rune) bool {
	return hangul.IsLeading(r)
}

func (l Layout) IsMedial(r rune) bool {
	return hangul.IsMedial(r)
}

func (l Layout) IsTrailing(r rune) bool {
	return hangul.IsTrailing(r)
}
