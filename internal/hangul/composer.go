package hangul

import (
	"bytes"

	logotype "github.com/gg582/hangul-logotype/hangul"
)

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
	leading  []rune
	vowels   []rune
	trailing []rune
}

func NewHangulComposer() *HangulComposer {
	return &HangulComposer{
		leading:  make([]rune, 0, 2),
		vowels:   make([]rune, 0, 3),
		trailing: make([]rune, 0, 2),
	}
}

var (
	choList  = []rune{'ㄱ', 'ㄲ', 'ㄴ', 'ㄷ', 'ㄸ', 'ㄹ', 'ㅁ', 'ㅂ', 'ㅃ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅉ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
	jungList = []rune{'ㅏ', 'ㅐ', 'ㅑ', 'ㅒ', 'ㅓ', 'ㅔ', 'ㅕ', 'ㅖ', 'ㅗ', 'ㅘ', 'ㅙ', 'ㅚ', 'ㅛ', 'ㅜ', 'ㅝ', 'ㅞ', 'ㅟ', 'ㅠ', 'ㅡ', 'ㅢ', 'ㅣ'}
	jongList = []rune{0, 'ㄱ', 'ㄲ', 'ㄳ', 'ㄴ', 'ㄵ', 'ㄶ', 'ㄷ', 'ㄹ', 'ㄺ', 'ㄻ', 'ㄼ', 'ㄽ', 'ㄾ', 'ㄿ', 'ㅀ', 'ㅁ', 'ㅂ', 'ㅄ', 'ㅅ', 'ㅆ', 'ㅇ', 'ㅈ', 'ㅊ', 'ㅋ', 'ㅌ', 'ㅍ', 'ㅎ'}
)

var doubleInitial = map[[2]rune]rune{
	{'ㄱ', 'ㄱ'}: 'ㄲ',
	{'ㄷ', 'ㄷ'}: 'ㄸ',
	{'ㅂ', 'ㅂ'}: 'ㅃ',
	{'ㅈ', 'ㅈ'}: 'ㅉ',
	{'ㅅ', 'ㅅ'}: 'ㅆ',
}

var (
	consonantSet = buildSet(append(append([]rune{}, choList...), filterZero(jongList)...))
	vowelSet     = buildSet(jungList)
	finalSet     = buildSet(filterZero(jongList))
)

func buildSet(list []rune) map[rune]struct{} {
	out := make(map[rune]struct{}, len(list))
	for _, r := range list {
		out[r] = struct{}{}
	}
	return out
}

func filterZero(list []rune) []rune {
	var out []rune
	for _, r := range list {
		if r != 0 {
			out = append(out, r)
		}
	}
	return out
}

func isConsonant(r rune) bool {
	_, ok := consonantSet[r]
	return ok
}

func isVowel(r rune) bool {
	_, ok := vowelSet[r]
	return ok
}

func (c *HangulComposer) Feed(ch rune, role JamoRole) CompositionResult {
	var commit []rune
	if isVowel(ch) {
		commit = c.handleVowel(ch)
	} else {
		commit = c.handleConsonant(ch, role)
	}
	return CompositionResult{
		Commit:  string(commit),
		Preedit: string(c.currentPreedit()),
	}
}

func (c *HangulComposer) Backspace() (string, bool) {
	switch {
	case len(c.trailing) > 0:
		c.trailing = c.trailing[:len(c.trailing)-1]
	case len(c.vowels) > 0:
		c.vowels = c.vowels[:len(c.vowels)-1]
	case len(c.leading) > 0:
		c.leading = c.leading[:len(c.leading)-1]
	default:
		return "", false
	}
	return string(c.currentPreedit()), true
}

func (c *HangulComposer) Flush() string {
	commit := string(c.composeCurrent())
	c.reset()
	return commit
}

func (c *HangulComposer) handleConsonant(ch rune, role JamoRole) []rune {
	var commit []rune

	if !isConsonant(ch) {
		commit = append(commit, c.composeCurrent()...)
		c.reset()
		return append(commit, ch)
	}

	if len(c.leading) == 0 {
		if len(c.vowels) > 0 {
			commit = append(commit, c.composeCurrent()...)
			c.reset()
		}
		c.leading = append(c.leading[:0], ch)
		c.trailing = c.trailing[:0]
		return commit
	}

	if role == RoleLeading {
		commit = append(commit, c.composeCurrent()...)
		c.reset()
		c.leading = append(c.leading[:0], ch)
		return commit
	}

	if len(c.vowels) == 0 {
		if len(c.leading) == 1 {
			pair := [2]rune{c.leading[0], ch}
			if _, ok := doubleInitial[pair]; ok {
				c.leading = append(c.leading, ch)
				return commit
			}
		}
		commit = append(commit, composeLeading(c.leading)...)
		c.leading = append(c.leading[:0], ch)
		c.trailing = c.trailing[:0]
		return commit
	}

	if role == RoleTrailing {
		return c.attachTrailing(ch)
	}

	if len(c.trailing) == 0 {
		c.trailing = append(c.trailing[:0], ch)
		return commit
	}

	candidate := append(append([]rune{}, c.trailing...), ch)
	if isValidFinal(candidate) {
		c.trailing = candidate
		return commit
	}

	commit = append(commit, c.composeCurrent()...)
	c.reset()
	c.leading = append(c.leading[:0], ch)
	return commit
}

func (c *HangulComposer) handleVowel(ch rune) []rune {
	var commit []rune

	if len(c.trailing) > 0 {
		commit = append(commit, c.detachTrailingForLeading()...)
	}

	if len(c.vowels) == 0 {
		c.vowels = append(c.vowels[:0], ch)
		return commit
	}

	candidate := append(append([]rune{}, c.vowels...), ch)
	if isCombinedVowel(candidate) {
		c.vowels = append(c.vowels, ch)
		return commit
	}

	if len(c.leading) == 0 {
		commit = append(commit, composeRunes(c.vowels)...)
	} else {
		commit = append(commit, c.composeCurrent()...)
	}

	c.leading = c.leading[:0]
	c.vowels = append(c.vowels[:0], ch)
	c.trailing = c.trailing[:0]
	return commit
}

func (c *HangulComposer) attachTrailing(ch rune) []rune {
	var commit []rune

	if !isConsonant(ch) {
		commit = append(commit, c.composeCurrent()...)
		c.reset()
		c.leading = append(c.leading[:0], ch)
		return commit
	}

	if len(c.trailing) == 0 {
		c.trailing = append(c.trailing[:0], ch)
		return commit
	}

	candidate := append(append([]rune{}, c.trailing...), ch)
	if isValidFinal(candidate) {
		c.trailing = candidate
		return commit
	}

	commit = append(commit, c.composeCurrent()...)
	c.reset()
	c.leading = append(c.leading[:0], ch)
	return commit
}

func (c *HangulComposer) detachTrailingForLeading() []rune {
	carry := c.trailing
	nextLeading := carry[len(carry)-1]
	if len(carry) > 1 {
		c.trailing = carry[:len(carry)-1]
	} else {
		c.trailing = c.trailing[:0]
	}
	committed := c.composeCurrent()
	c.leading = append(c.leading[:0], nextLeading)
	c.vowels = c.vowels[:0]
	c.trailing = c.trailing[:0]
	return committed
}

func (c *HangulComposer) composeCurrent() []rune {
	raw := make([]rune, 0, 1+len(c.vowels)+len(c.trailing))
	if lead := composeLeading(c.leading); len(lead) > 0 {
		raw = append(raw, lead...)
	}
	raw = append(raw, c.vowels...)
	raw = append(raw, c.trailing...)
	return composeRunes(raw)
}

func (c *HangulComposer) currentPreedit() []rune {
	return c.composeCurrent()
}

func (c *HangulComposer) reset() {
	c.leading = c.leading[:0]
	c.vowels = c.vowels[:0]
	c.trailing = c.trailing[:0]
}

func composeLeading(parts []rune) []rune {
	switch len(parts) {
	case 0:
		return nil
	case 1:
		return []rune{parts[0]}
	case 2:
		pair := [2]rune{parts[0], parts[1]}
		if combined, ok := doubleInitial[pair]; ok {
			return []rune{combined}
		}
		return []rune{parts[0], parts[1]}
	default:
		return append([]rune{}, parts...)
	}
}

func isCombinedVowel(candidate []rune) bool {
	composed := composeRunes(candidate)
	return len(composed) == 1 && isVowel(composed[0])
}

func isValidFinal(candidate []rune) bool {
	composed := composeRunes(candidate)
	if len(composed) != 1 {
		return false
	}
	_, ok := finalSet[composed[0]]
	return ok
}

func composeRunes(raw []rune) []rune {
	if len(raw) == 0 {
		return nil
	}
	var buf bytes.Buffer
	logotype.LogoType(&buf, raw)
	return []rune(buf.String())
}
