package hangul

type JamoRole int

const (
	RoleAuto JamoRole = iota
	RoleLeading
	RoleTrailing
)

type CompositionResult struct {
	Commit  string
	Preedit string
}

type HangulComposer struct {
	leading  *rune
	vowel    *rune
	trailing *rune
}

func NewHangulComposer() *HangulComposer {
	return &HangulComposer{}
}

func runePtr(r rune) *rune {
	v := r
	return &v
}

var (
	choList  = []rune{'ㄱ', 'ㄲ', 'ㄴ', 'ㄷ', 'ㄸ', 'ㄹ', 'ㅁ', 'ㅂ', 'ㅃ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅉ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
	jungList = []rune{'ㅏ', 'ㅐ', 'ㅑ', 'ㅒ', 'ㅓ', 'ㅔ', 'ㅕ', 'ㅖ', 'ㅗ', 'ㅘ', 'ㅙ', 'ㅚ', 'ㅛ', 'ㅜ', 'ㅝ', 'ㅞ', 'ㅟ', 'ㅠ', 'ㅡ', 'ㅢ', 'ㅣ'}
	jongList = []rune{0, 'ㄱ', 'ㄲ', 'ㄳ', 'ㄴ', 'ㄵ', 'ㄶ', 'ㄷ', 'ㄹ', 'ㄺ', 'ㄻ', 'ㄼ', 'ㄽ', 'ㄾ', 'ㄿ', 'ㅀ', 'ㅁ', 'ㅂ', 'ㅄ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
)

var (
	doubleInitial = map[[2]rune]rune{
		{'ㄱ', 'ㄱ'}: 'ㄲ',
		{'ㄷ', 'ㄷ'}: 'ㄸ',
		{'ㅂ', 'ㅂ'}: 'ㅃ',
		{'ㅈ', 'ㅈ'}: 'ㅉ',
		{'ㅅ', 'ㅅ'}: 'ㅆ',
	}
	doubleMedial = map[[2]rune]rune{
		{'ㅗ', 'ㅏ'}: 'ㅘ',
		{'ㅗ', 'ㅐ'}: 'ㅙ',
		{'ㅗ', 'ㅣ'}: 'ㅚ',
		{'ㅜ', 'ㅓ'}: 'ㅝ',
		{'ㅜ', 'ㅔ'}: 'ㅞ',
		{'ㅜ', 'ㅣ'}: 'ㅟ',
		{'ㅡ', 'ㅣ'}: 'ㅢ',
	}
	doubleFinal = map[[2]rune]rune{
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
	initialDecompose = invertDouble(doubleInitial)
	medialDecompose  = invertDouble(doubleMedial)
	finalDecompose   = invertDouble(doubleFinal)
)

var (
	choseongIndex  = buildIndex(choList)
	jungseongIndex = buildIndex(jungList)
	jongseongIndex = buildIndex(jongList)
)

var (
	consonantSet = buildSet(append(append([]rune{}, choList...), filterZero(jongList)...))
	vowelSet     = buildSet(jungList)
)

func invertDouble(src map[[2]rune]rune) map[rune][2]rune {
	dst := make(map[rune][2]rune, len(src))
	for pair, value := range src {
		dst[value] = pair
	}
	return dst
}

func buildIndex(list []rune) map[rune]int {
	idx := make(map[rune]int, len(list))
	for i, ch := range list {
		idx[ch] = i
	}
	return idx
}

func buildSet(list []rune) map[rune]struct{} {
	set := make(map[rune]struct{}, len(list))
	for _, ch := range list {
		set[ch] = struct{}{}
	}
	return set
}

func filterZero(list []rune) []rune {
	out := make([]rune, 0, len(list))
	for _, ch := range list {
		if ch != 0 {
			out = append(out, ch)
		}
	}
	return out
}

func isConsonant(ch rune) bool {
	_, ok := consonantSet[ch]
	return ok
}

func isVowel(ch rune) bool {
	_, ok := vowelSet[ch]
	return ok
}

func (c *HangulComposer) Feed(ch rune, role JamoRole) CompositionResult {
	var commit []rune
	if isVowel(ch) {
		commit = c.handleVowel(ch)
	} else {
		commit = c.handleConsonant(ch, role)
	}
	result := CompositionResult{
		Commit:  string(commit),
		Preedit: string(c.currentPreedit()),
	}
	return result
}

func (c *HangulComposer) Backspace() (string, bool) {
	if c.trailing != nil {
		if pair, ok := finalDecompose[*c.trailing]; ok {
			first := pair[0]
			c.trailing = runePtr(first)
		} else {
			c.trailing = nil
		}
		return string(c.currentPreedit()), true
	}
	if c.vowel != nil {
		if pair, ok := medialDecompose[*c.vowel]; ok {
			first := pair[0]
			c.vowel = runePtr(first)
		} else {
			c.vowel = nil
			if c.leading != nil && *c.leading == 'ㅇ' {
				c.leading = nil
			}
		}
		return string(c.currentPreedit()), true
	}
	if c.leading != nil {
		if pair, ok := initialDecompose[*c.leading]; ok {
			first := pair[0]
			c.leading = runePtr(first)
		} else {
			c.leading = nil
		}
		return string(c.currentPreedit()), true
	}
	return "", false
}

func (c *HangulComposer) Flush() string {
	commit := string(c.compose())
	c.leading = nil
	c.vowel = nil
	c.trailing = nil
	return commit
}

func (c *HangulComposer) handleConsonant(ch rune, role JamoRole) []rune {
	var commit []rune
	forceTrailing := role == RoleTrailing
	forceLeading := role == RoleLeading

	if c.leading == nil {
		c.leading = runePtr(ch)
		c.trailing = nil
		return commit
	}

	if forceLeading {
		commit = c.compose()
		c.leading = runePtr(ch)
		c.vowel = nil
		c.trailing = nil
		return commit
	}

	if c.vowel == nil {
		pair := [2]rune{*c.leading, ch}
		if combined, ok := doubleInitial[pair]; ok {
			c.leading = runePtr(combined)
		} else {
			commit = append(commit, *c.leading)
			c.leading = runePtr(ch)
		}
		return commit
	}

	if forceTrailing {
		return c.attachTrailing(ch)
	}

	if c.trailing == nil {
		if isConsonant(ch) {
			c.trailing = runePtr(ch)
			return commit
		}
		commit = c.compose()
		c.leading = runePtr(ch)
		c.vowel = nil
		c.trailing = nil
		return commit
	}

	pair := [2]rune{*c.trailing, ch}
	if combined, ok := doubleFinal[pair]; ok {
		c.trailing = runePtr(combined)
	} else {
		commit = c.compose()
		c.leading = runePtr(ch)
		c.vowel = nil
		c.trailing = nil
	}
	return commit
}

func (c *HangulComposer) handleVowel(ch rune) []rune {
	var commit []rune
	if c.leading == nil {
		c.leading = runePtr('ㅇ')
	}

	if c.vowel == nil {
		c.vowel = runePtr(ch)
		return commit
	}

	pair := [2]rune{*c.vowel, ch}
	if combined, ok := doubleMedial[pair]; ok {
		c.vowel = runePtr(combined)
		return commit
	}

	if c.trailing != nil {
		if split, ok := finalDecompose[*c.trailing]; ok {
			first := split[0]
			second := split[1]
			c.trailing = runePtr(first)
			commit = c.compose()
			c.leading = runePtr(second)
			c.vowel = runePtr(ch)
			c.trailing = nil
			return commit
		}
		trailing := *c.trailing
		c.trailing = nil
		commit = c.compose()
		c.leading = runePtr(trailing)
		c.vowel = runePtr(ch)
		return commit
	}

	commit = c.compose()
	c.leading = runePtr('ㅇ')
	c.vowel = runePtr(ch)
	c.trailing = nil
	return commit
}

func (c *HangulComposer) attachTrailing(ch rune) []rune {
	var commit []rune
	if c.trailing == nil {
		if isConsonant(ch) {
			c.trailing = runePtr(ch)
			return commit
		}
		commit = c.compose()
		c.leading = runePtr(ch)
		c.vowel = nil
		c.trailing = nil
		return commit
	}

	pair := [2]rune{*c.trailing, ch}
	if combined, ok := doubleFinal[pair]; ok {
		c.trailing = runePtr(combined)
		return commit
	}

	commit = c.compose()
	c.leading = runePtr(ch)
	c.vowel = nil
	c.trailing = nil
	return commit
}

func (c *HangulComposer) compose() []rune {
	if c.leading == nil && c.vowel == nil {
		return []rune{}
	}
	if c.leading != nil && c.vowel != nil {
		leadIdx := choseongIndex[*c.leading]
		vowelIdx := jungseongIndex[*c.vowel]
		tailIdx := 0
		if c.trailing != nil {
			tailIdx = jongseongIndex[*c.trailing]
		}
		codepoint := rune(0xAC00 + ((leadIdx*21)+vowelIdx)*28 + tailIdx)
		return []rune{codepoint}
	}
	if c.leading != nil {
		return []rune{*c.leading}
	}
	if c.vowel != nil {
		return []rune{*c.vowel}
	}
	return []rune{}
}

func (c *HangulComposer) currentPreedit() []rune {
	return c.compose()
}
