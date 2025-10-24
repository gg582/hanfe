package automata

// Composer replicates the fcitx/libhangul automata flow, keeping track of the
// leading, medial, and trailing jamo while keys are processed.
type Composer struct {
	lead  rune
	vowel rune
	tail  rune
}

func NewComposer() *Composer {
	return &Composer{}
}

func (c *Composer) Type(key rune) (string, string) {
	entry, ok := lookupKey(key)
	if !ok {
		committed := c.Flush()
		return committed + string(key), ""
	}

	if entry.vowel != 0 {
		return c.handleVowel(entry.vowel)
	}

	if entry.lead != 0 {
		return c.handleConsonant(entry.lead, entry.tail)
	}

	return "", c.preedit()
}

func (c *Composer) Backspace() (string, bool) {
	if c.tail != 0 {
		if pair, ok := finalSplit[c.tail]; ok {
			c.tail = pair[0]
		} else {
			c.tail = 0
		}
		return c.preedit(), true
	}

	if c.vowel != 0 {
		if pair, ok := medialSplit[c.vowel]; ok {
			c.vowel = pair[0]
		} else {
			c.vowel = 0
		}
		return c.preedit(), true
	}

	if c.lead != 0 {
		if pair, ok := initialSplit[c.lead]; ok {
			c.lead = pair[0]
		} else {
			c.lead = 0
		}
		return c.preedit(), true
	}

	return "", false
}

func (c *Composer) Flush() string {
	if c.lead == 0 && c.vowel == 0 {
		return ""
	}

	committed := c.preedit()
	c.lead = 0
	c.vowel = 0
	c.tail = 0
	return committed
}

func (c *Composer) handleConsonant(lead rune, tail rune) (string, string) {
	var commit []rune

	if c.vowel == 0 {
		if c.lead == 0 {
			c.lead = lead
			return "", c.preedit()
		}
		if combined, ok := initialCompose[[2]rune{c.lead, lead}]; ok {
			c.lead = combined
			return "", c.preedit()
		}

		commit = append(commit, []rune(c.preedit())...)
		c.lead = lead
		c.vowel = 0
		c.tail = 0
		return string(commit), c.preedit()
	}

	if tail == 0 {
		commit = append(commit, []rune(c.preedit())...)
		c.lead = lead
		c.vowel = 0
		c.tail = 0
		return string(commit), c.preedit()
	}

	if c.tail == 0 {
		c.tail = tail
		return "", c.preedit()
	}

	if combined, ok := finalCompose[[2]rune{c.tail, tail}]; ok {
		c.tail = combined
		return "", c.preedit()
	}

	commit = append(commit, []rune(c.preedit())...)
	c.lead = lead
	c.vowel = 0
	c.tail = 0
	return string(commit), c.preedit()
}

func (c *Composer) handleVowel(vowel rune) (string, string) {
	var commit []rune

	if c.vowel == 0 {
		if c.lead == 0 {
			c.vowel = vowel
			return "", c.preedit()
		}
		c.vowel = vowel
		return "", c.preedit()
	}

	if c.tail != 0 {
		carry := c.tail
		if pair, ok := finalSplit[c.tail]; ok {
			c.tail = pair[0]
			carry = pair[1]
			commit = append(commit, compose(c.lead, c.vowel, c.tail))
			c.lead = toLeadFromTail(carry)
			c.vowel = vowel
			c.tail = 0
			return string(commit), c.preedit()
		}

		commit = append(commit, compose(c.lead, c.vowel, 0))
		c.lead = toLeadFromTail(carry)
		c.vowel = vowel
		c.tail = 0
		return string(commit), c.preedit()
	}

	if combined, ok := medialCompose[[2]rune{c.vowel, vowel}]; ok {
		c.vowel = combined
		return "", c.preedit()
	}

	commit = append(commit, compose(c.lead, c.vowel, 0))
	c.lead = 0
	c.vowel = vowel
	c.tail = 0
	return string(commit), c.preedit()
}

func (c *Composer) preedit() string {
	if c.vowel == 0 {
		if c.lead == 0 {
			return ""
		}
		return string(c.lead)
	}

	if c.lead == 0 {
		return string(c.vowel)
	}

	return string(compose(c.lead, c.vowel, c.tail))
}

func compose(lead rune, vowel rune, tail rune) rune {
	li, ok := choseongIndex[lead]
	if !ok {
		if lead == 0 {
			return vowel
		}
		return lead
	}
	mi, ok := jungseongIndex[vowel]
	if !ok {
		return vowel
	}
	ti := 0
	if tail != 0 {
		if idx, ok := jongseongIndex[tail]; ok {
			ti = idx
		}
	}

	base := 0xAC00
	composed := rune(base + li*21*28 + mi*28 + ti)
	return composed
}
