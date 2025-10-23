package ime

import (
	"strings"

	"github.com/suapapa/go_hangul/hangul"
	"github.com/suapapa/go_hangul/keyboard"
)

type Composer struct {
	layout keyboard.Layout
	lead   rune
	vowel  rune
	tail   rune
	text   []rune
}

func NewComposer(layout keyboard.Layout) *Composer {
	return &Composer{layout: layout, text: make([]rune, 0, 32)}
}

func (c *Composer) TypeKey(key rune) bool {
	if j, ok := c.layout.ConsonantForKey(key); ok {
		c.processConsonant(j)
		return true
	}
	if j, ok := c.layout.VowelForKey(key); ok {
		c.processVowel(j)
		return true
	}
	return false
}

func (c *Composer) AppendLiteral(r rune) {
	c.commitSyllable()
	c.text = append(c.text, r)
}

func (c *Composer) Space() {
	c.AppendLiteral(' ')
}

func (c *Composer) Backspace() {
	if c.tail != 0 {
		if base, _, ok := c.layout.DecomposeFinal(c.tail); ok {
			c.tail = base
			return
		}
		c.tail = 0
		return
	}
	if c.vowel != 0 {
		if base, _, ok := c.layout.DecomposeMedial(c.vowel); ok {
			c.vowel = base
			return
		}
		c.vowel = 0
		return
	}
	if c.lead != 0 {
		c.lead = 0
		return
	}
	if len(c.text) > 0 {
		c.text = c.text[:len(c.text)-1]
	}
}

func (c *Composer) Enter() string {
	line := c.FlushText()
	c.text = make([]rune, 0, 32)
	return line
}

func (c *Composer) FlushText() string {
	c.commitSyllable()
	result := string(c.text)
	return result
}

func (c *Composer) Reset() {
	c.lead, c.vowel, c.tail = 0, 0, 0
	c.text = c.text[:0]
}

func (c *Composer) Text() string {
	builder := strings.Builder{}
	builder.Grow(len(c.text) + 4)
	builder.WriteString(string(c.text))
	if composed, ok := c.currentSyllable(); ok {
		builder.WriteRune(composed)
	} else {
		if c.lead != 0 {
			builder.WriteRune(c.lead)
		}
		if c.vowel != 0 {
			builder.WriteRune(c.vowel)
		}
		if c.tail != 0 {
			builder.WriteRune(c.tail)
		}
	}
	return builder.String()
}

func (c *Composer) processConsonant(j rune) {
	if c.lead == 0 {
		c.lead = j
		return
	}

	if c.vowel == 0 {
		c.commitSyllable()
		c.lead = j
		return
	}

	if c.tail == 0 {
		c.tail = j
		return
	}

	if combined, ok := c.layout.CombineFinal(c.tail, j); ok {
		c.tail = combined
		return
	}

	c.commitSyllable()
	c.lead = j
}

func (c *Composer) processVowel(j rune) {
	if c.lead == 0 {
		c.lead = c.layout.SilentLeading
	}

	if c.vowel == 0 {
		c.vowel = j
		return
	}

	if combined, ok := c.layout.CombineMedial(c.vowel, j); ok {
		c.vowel = combined
		return
	}

	if c.tail != 0 {
		if first, second, ok := c.layout.DecomposeFinal(c.tail); ok {
			c.tail = first
			c.commitSyllable()
			c.lead = second
			c.vowel = j
			c.tail = 0
			return
		}
		previous := c.tail
		c.tail = 0
		c.commitSyllable()
		c.lead = previous
		c.vowel = j
		return
	}

	c.commitSyllable()
	c.lead = c.layout.SilentLeading
	c.vowel = j
}

func (c *Composer) currentSyllable() (rune, bool) {
	if c.lead == 0 || c.vowel == 0 {
		return 0, false
	}
	r, ok := hangul.Compose(c.lead, c.vowel, c.tail)
	if !ok {
		return 0, false
	}
	return r, true
}

func (c *Composer) commitSyllable() {
	if c.lead == 0 && c.vowel == 0 && c.tail == 0 {
		return
	}
	if r, ok := c.currentSyllable(); ok {
		c.text = append(c.text, r)
	} else {
		if c.lead != 0 {
			c.text = append(c.text, c.lead)
		}
		if c.vowel != 0 {
			c.text = append(c.text, c.vowel)
		}
		if c.tail != 0 {
			c.text = append(c.text, c.tail)
		}
	}
	c.lead, c.vowel, c.tail = 0, 0, 0
}
