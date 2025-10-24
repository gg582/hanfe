package automata

var (
	choseong  = []rune{'ㄱ', 'ㄲ', 'ㄴ', 'ㄷ', 'ㄸ', 'ㄹ', 'ㅁ', 'ㅂ', 'ㅃ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅉ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
	jungseong = []rune{'ㅏ', 'ㅐ', 'ㅑ', 'ㅒ', 'ㅓ', 'ㅔ', 'ㅕ', 'ㅖ', 'ㅗ', 'ㅘ', 'ㅙ', 'ㅚ', 'ㅛ', 'ㅜ', 'ㅝ', 'ㅞ', 'ㅟ', 'ㅠ', 'ㅡ', 'ㅢ', 'ㅣ'}
	jongseong = []rune{0, 'ㄱ', 'ㄲ', 'ㄳ', 'ㄴ', 'ㄵ', 'ㄶ', 'ㄷ', 'ㄹ', 'ㄺ', 'ㄻ', 'ㄼ', 'ㄽ', 'ㄾ', 'ㄿ', 'ㅀ', 'ㅁ', 'ㅂ', 'ㅄ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
)

var (
	initialCompose = map[[2]rune]rune{
		{'ㄱ', 'ㄱ'}: 'ㄲ',
		{'ㄷ', 'ㄷ'}: 'ㄸ',
		{'ㅂ', 'ㅂ'}: 'ㅃ',
		{'ㅈ', 'ㅈ'}: 'ㅉ',
		{'ㅅ', 'ㅅ'}: 'ㅆ',
	}
	medialCompose = map[[2]rune]rune{
		{'ㅗ', 'ㅏ'}: 'ㅘ',
		{'ㅗ', 'ㅐ'}: 'ㅙ',
		{'ㅗ', 'ㅣ'}: 'ㅚ',
		{'ㅜ', 'ㅓ'}: 'ㅝ',
		{'ㅜ', 'ㅔ'}: 'ㅞ',
		{'ㅜ', 'ㅣ'}: 'ㅟ',
		{'ㅡ', 'ㅣ'}: 'ㅢ',
	}
	finalCompose = map[[2]rune]rune{
		{'ㄱ', 'ㄱ'}: 'ㄲ',
		{'ㄱ', 'ㅅ'}: 'ㄳ',
		{'ㄴ', 'ㅈ'}: 'ㄵ',
		{'ㄴ', 'ㅎ'}: 'ㄶ',
		{'ㄹ', 'ㄱ'}: 'ㄺ',
		{'ㄹ', 'ㅁ'}: 'ㄻ',
		{'ㄹ', 'ㅂ'}: 'ㄼ',
		{'ㄹ', 'ㅅ'}: 'ㄽ',
		{'ㄹ', 'ㅌ'}: 'ㄾ',
		{'ㄹ', 'ㅍ'}: 'ㄿ',
		{'ㄹ', 'ㅎ'}: 'ㅀ',
		{'ㅂ', 'ㅅ'}: 'ㅄ',
		{'ㅅ', 'ㅅ'}: 'ㅆ',
	}
)

var (
	initialSplit = invertDouble(initialCompose)
	medialSplit  = invertDouble(medialCompose)
	finalSplit   = invertDouble(finalCompose)
)

var (
	choseongIndex  = buildIndex(choseong)
	jungseongIndex = buildIndex(jungseong)
	jongseongIndex = buildIndex(jongseong)
)

var tailToLead = map[rune]rune{
	'ㄱ': 'ㄱ',
	'ㄲ': 'ㄲ',
	'ㄳ': 'ㄱ',
	'ㄴ': 'ㄴ',
	'ㄵ': 'ㄴ',
	'ㄶ': 'ㄴ',
	'ㄷ': 'ㄷ',
	'ㄹ': 'ㄹ',
	'ㄺ': 'ㄹ',
	'ㄻ': 'ㄹ',
	'ㄼ': 'ㄹ',
	'ㄽ': 'ㄹ',
	'ㄾ': 'ㄹ',
	'ㄿ': 'ㄹ',
	'ㅀ': 'ㄹ',
	'ㅁ': 'ㅁ',
	'ㅂ': 'ㅂ',
	'ㅄ': 'ㅂ',
	'ㅅ': 'ㅅ',
	'ㅆ': 'ㅆ',
	'ㅇ': 'ㅇ',
	'ㅈ': 'ㅈ',
	'ㅊ': 'ㅊ',
	'ㅋ': 'ㅋ',
	'ㅌ': 'ㅌ',
	'ㅍ': 'ㅍ',
	'ㅎ': 'ㅎ',
}

func toLeadFromTail(t rune) rune {
	if lead, ok := tailToLead[t]; ok {
		return lead
	}
	return t
}

func invertDouble(src map[[2]rune]rune) map[rune][2]rune {
	dst := make(map[rune][2]rune, len(src))
	for pair, v := range src {
		dst[v] = pair
	}
	return dst
}

func buildIndex(list []rune) map[rune]int {
	idx := make(map[rune]int, len(list))
	for i, r := range list {
		idx[r] = i
	}
	return idx
}
