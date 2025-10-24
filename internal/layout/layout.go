package layout

import (
	"fmt"
	"sort"

	"hanfe/internal/hangul"
	"hanfe/internal/linux"
)

type SymbolKind int

const (
	SymbolPassthrough SymbolKind = iota
	SymbolText
	SymbolJamo
)

type LayoutSymbol struct {
	Kind         SymbolKind
	Text         string
	Jamo         rune
	Role         hangul.JamoRole
	CommitBefore bool
}

type LayoutEntry struct {
	Normal  *LayoutSymbol
	Shifted *LayoutSymbol
}

type Layout struct {
	name    string
	mapping map[uint16]LayoutEntry
}

func NewLayout(name string) *Layout {
	return &Layout{name: name, mapping: make(map[uint16]LayoutEntry)}
}

func (l *Layout) Name() string { return l.name }

func (l *Layout) Translate(code uint16, shift bool) *LayoutSymbol {
	if l == nil {
		return nil
	}
	entry, ok := l.mapping[code]
	if !ok {
		return nil
	}
	if shift && entry.Shifted != nil {
		return entry.Shifted
	}
	if entry.Normal != nil {
		return entry.Normal
	}
	if entry.Shifted != nil {
		return entry.Shifted
	}
	return nil
}

func NewTextSymbol(value string) *LayoutSymbol {
	return &LayoutSymbol{Kind: SymbolText, Text: value}
}

func NewJamoSymbol(value rune, role hangul.JamoRole) *LayoutSymbol {
	return &LayoutSymbol{Kind: SymbolJamo, Jamo: value, Role: role}
}

func NewPassthroughSymbol(commitBefore bool) *LayoutSymbol {
	return &LayoutSymbol{Kind: SymbolPassthrough, CommitBefore: commitBefore}
}

func makeJamo(value rune) *LayoutSymbol { return NewJamoSymbol(value, hangul.RoleAuto) }

func makeJamoSymbol(value rune, role hangul.JamoRole) *LayoutSymbol {
	return NewJamoSymbol(value, role)
}

func makePassthroughSymbol(commitBefore bool) *LayoutSymbol {
	return NewPassthroughSymbol(commitBefore)
}

func addEntry(mapping map[uint16]LayoutEntry, key int, normal *LayoutSymbol, shifted *LayoutSymbol) {
	mapping[uint16(key)] = LayoutEntry{Normal: normal, Shifted: shifted}
}

func (l *Layout) ApplyOverride(code uint16, shift bool, symbol *LayoutSymbol) {
	if l == nil {
		return
	}
	entry := l.mapping[code]
	if shift {
		entry.Shifted = symbol
	} else {
		entry.Normal = symbol
	}
	l.mapping[code] = entry
}

func buildDubeolsik() *Layout {
	layout := NewLayout("dubeolsik")
	mapping := layout.mapping
	addEntry(mapping, linux.KeyQ, makeJamo('ㅂ'), makeJamo('ㅃ'))
	addEntry(mapping, linux.KeyW, makeJamo('ㅈ'), makeJamo('ㅉ'))
	addEntry(mapping, linux.KeyE, makeJamo('ㄷ'), makeJamo('ㄸ'))
	addEntry(mapping, linux.KeyR, makeJamo('ㄱ'), makeJamo('ㄲ'))
	addEntry(mapping, linux.KeyT, makeJamo('ㅅ'), makeJamo('ㅆ'))
	addEntry(mapping, linux.KeyY, makeJamo('ㅛ'), nil)
	addEntry(mapping, linux.KeyU, makeJamo('ㅕ'), nil)
	addEntry(mapping, linux.KeyI, makeJamo('ㅑ'), nil)
	addEntry(mapping, linux.KeyO, makeJamo('ㅐ'), makeJamo('ㅒ'))
	addEntry(mapping, linux.KeyP, makeJamo('ㅔ'), makeJamo('ㅖ'))
	addEntry(mapping, linux.KeyA, makeJamo('ㅁ'), nil)
	addEntry(mapping, linux.KeyS, makeJamo('ㄴ'), nil)
	addEntry(mapping, linux.KeyD, makeJamo('ㅇ'), nil)
	addEntry(mapping, linux.KeyF, makeJamo('ㄹ'), nil)
	addEntry(mapping, linux.KeyG, makeJamo('ㅎ'), nil)
	addEntry(mapping, linux.KeyH, makeJamo('ㅗ'), nil)
	addEntry(mapping, linux.KeyJ, makeJamo('ㅓ'), nil)
	addEntry(mapping, linux.KeyK, makeJamo('ㅏ'), nil)
	addEntry(mapping, linux.KeyL, makeJamo('ㅣ'), nil)

	passthrough := makePassthroughSymbol(true)
	addEntry(mapping, linux.KeySemicolon, passthrough, passthrough)
	addEntry(mapping, linux.KeyApostrophe, passthrough, passthrough)

	addEntry(mapping, linux.KeyZ, makeJamo('ㅋ'), nil)
	addEntry(mapping, linux.KeyX, makeJamo('ㅌ'), nil)
	addEntry(mapping, linux.KeyC, makeJamo('ㅊ'), nil)
	addEntry(mapping, linux.KeyV, makeJamo('ㅍ'), nil)
	addEntry(mapping, linux.KeyB, makeJamo('ㅠ'), nil)
	addEntry(mapping, linux.KeyN, makeJamo('ㅜ'), nil)
	addEntry(mapping, linux.KeyM, makeJamo('ㅡ'), nil)

	addEntry(mapping, linux.KeyComma, passthrough, passthrough)
	addEntry(mapping, linux.KeyDot, passthrough, passthrough)
	addEntry(mapping, linux.KeySlash, passthrough, passthrough)

	addEntry(mapping, linux.KeySpace, makePassthroughSymbol(true), nil)

	passthroughPairKeys := []int{
		linux.Key1, linux.Key2, linux.Key3, linux.Key4, linux.Key5,
		linux.Key6, linux.Key7, linux.Key8, linux.Key9, linux.Key0,
		linux.KeyMinus, linux.KeyEqual, linux.KeyLeftBrace, linux.KeyRightBrace,
		linux.KeyBackslash, linux.KeyGrave,
	}
	for _, key := range passthroughPairKeys {
		addEntry(mapping, key, passthrough, passthrough)
	}

	specialKeys := []int{linux.KeyTab, linux.KeyEnter, linux.KeyEsc, linux.KeyBackspace}
	for _, key := range specialKeys {
		addEntry(mapping, key, makePassthroughSymbol(true), nil)
	}

	return layout
}

func buildSebeolsik390() *Layout {
	layout := NewLayout("sebeolsik-390")
	mapping := layout.mapping
	passthrough := makePassthroughSymbol(true)

	passthroughKeys := []int{
		linux.KeyGrave,
		linux.Key1, linux.Key2, linux.Key3, linux.Key4, linux.Key5,
		linux.Key6, linux.Key7, linux.Key8, linux.Key9, linux.Key0,
		linux.KeyMinus, linux.KeyEqual,
	}
	for _, key := range passthroughKeys {
		addEntry(mapping, key, passthrough, passthrough)
	}

	addEntry(mapping, linux.KeyQ, makeJamo('ㅂ'), makeJamo('ㅃ'))
	addEntry(mapping, linux.KeyW, makeJamo('ㅈ'), makeJamo('ㅉ'))
	addEntry(mapping, linux.KeyE, makeJamo('ㄷ'), makeJamo('ㄸ'))
	addEntry(mapping, linux.KeyR, makeJamo('ㄱ'), makeJamo('ㄲ'))
	addEntry(mapping, linux.KeyT, makeJamo('ㅅ'), makeJamo('ㅆ'))
	addEntry(mapping, linux.KeyY, makeJamo('ㅛ'), makeJamoSymbol('ㅅ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyU, makeJamo('ㅕ'), makeJamoSymbol('ㅈ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyI, makeJamo('ㅑ'), makeJamoSymbol('ㅊ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyO, makeJamo('ㅐ'), makeJamoSymbol('ㅋ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyP, makeJamo('ㅔ'), makeJamoSymbol('ㅌ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyLeftBrace, makeJamo('ㅒ'), makeJamoSymbol('ㅍ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyRightBrace, makeJamo('ㅖ'), makeJamoSymbol('ㅎ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyBackslash, makeJamo('ㅢ'), passthrough)

	addEntry(mapping, linux.KeyA, makeJamo('ㅁ'), makeJamo('ㅁ'))
	addEntry(mapping, linux.KeyS, makeJamo('ㄴ'), makeJamo('ㄴ'))
	addEntry(mapping, linux.KeyD, makeJamo('ㅇ'), makeJamo('ㅇ'))
	addEntry(mapping, linux.KeyF, makeJamo('ㄹ'), makeJamo('ㄹ'))
	addEntry(mapping, linux.KeyG, makeJamo('ㅎ'), makeJamo('ㅎ'))
	addEntry(mapping, linux.KeyH, makeJamo('ㅗ'), makeJamoSymbol('ㄱ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyJ, makeJamo('ㅓ'), makeJamoSymbol('ㄴ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyK, makeJamo('ㅏ'), makeJamoSymbol('ㄷ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyL, makeJamo('ㅣ'), makeJamoSymbol('ㄹ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeySemicolon, makeJamo('ㅠ'), makeJamoSymbol('ㅁ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyApostrophe, makeJamo('ㅜ'), makeJamoSymbol('ㅂ', hangul.RoleTrailing))

	addEntry(mapping, linux.KeyZ, makeJamo('ㅋ'), makeJamo('ㅋ'))
	addEntry(mapping, linux.KeyX, makeJamo('ㅌ'), makeJamo('ㅌ'))
	addEntry(mapping, linux.KeyC, makeJamo('ㅊ'), makeJamo('ㅊ'))
	addEntry(mapping, linux.KeyV, makeJamo('ㅍ'), makeJamo('ㅍ'))
	addEntry(mapping, linux.KeyB, makeJamo('ㅠ'), makeJamoSymbol('ㅇ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyN, makeJamo('ㅜ'), makeJamoSymbol('ㅅ', hangul.RoleTrailing))
	addEntry(mapping, linux.KeyM, makeJamo('ㅡ'), makeJamoSymbol('ㅎ', hangul.RoleTrailing))

	addEntry(mapping, linux.KeyComma, makeJamo('ㅘ'), makeJamo('ㅙ'))
	addEntry(mapping, linux.KeyDot, makeJamo('ㅝ'), makeJamo('ㅞ'))
	addEntry(mapping, linux.KeySlash, makeJamo('ㅟ'), passthrough)

	addEntry(mapping, linux.KeySpace, makePassthroughSymbol(true), nil)

	specialKeys := []int{linux.KeyEnter, linux.KeyTab, linux.KeyEsc, linux.KeyBackspace}
	for _, key := range specialKeys {
		addEntry(mapping, key, makePassthroughSymbol(true), nil)
	}

	return layout
}

func buildLatin() *Layout {
	layout := NewLayout("latin")
	mapping := layout.mapping

	passthrough := NewPassthroughSymbol(true)

	letters := []struct {
		key   int
		lower string
		upper string
	}{
		{linux.KeyA, "a", "A"},
		{linux.KeyB, "b", "B"},
		{linux.KeyC, "c", "C"},
		{linux.KeyD, "d", "D"},
		{linux.KeyE, "e", "E"},
		{linux.KeyF, "f", "F"},
		{linux.KeyG, "g", "G"},
		{linux.KeyH, "h", "H"},
		{linux.KeyI, "i", "I"},
		{linux.KeyJ, "j", "J"},
		{linux.KeyK, "k", "K"},
		{linux.KeyL, "l", "L"},
		{linux.KeyM, "m", "M"},
		{linux.KeyN, "n", "N"},
		{linux.KeyO, "o", "O"},
		{linux.KeyP, "p", "P"},
		{linux.KeyQ, "q", "Q"},
		{linux.KeyR, "r", "R"},
		{linux.KeyS, "s", "S"},
		{linux.KeyT, "t", "T"},
		{linux.KeyU, "u", "U"},
		{linux.KeyV, "v", "V"},
		{linux.KeyW, "w", "W"},
		{linux.KeyX, "x", "X"},
		{linux.KeyY, "y", "Y"},
		{linux.KeyZ, "z", "Z"},
	}
	for _, entry := range letters {
		addEntry(mapping, entry.key, NewTextSymbol(entry.lower), NewTextSymbol(entry.upper))
	}
	digits := []struct {
		key   int
		value string
		shift string
	}{
		{linux.Key1, "1", "!"},
		{linux.Key2, "2", "@"},
		{linux.Key3, "3", "#"},
		{linux.Key4, "4", "$"},
		{linux.Key5, "5", "%"},
		{linux.Key6, "6", "^"},
		{linux.Key7, "7", "&"},
		{linux.Key8, "8", "*"},
		{linux.Key9, "9", "("},
		{linux.Key0, "0", ")"},
	}
	for _, entry := range digits {
		addEntry(mapping, entry.key, NewTextSymbol(entry.value), NewTextSymbol(entry.shift))
	}

	symbols := map[int][2]*LayoutSymbol{
		linux.KeyMinus:      {NewTextSymbol("-"), NewTextSymbol("_")},
		linux.KeyEqual:      {NewTextSymbol("="), NewTextSymbol("+")},
		linux.KeyLeftBrace:  {NewTextSymbol("["), NewTextSymbol("{")},
		linux.KeyRightBrace: {NewTextSymbol("]"), NewTextSymbol("}")},
		linux.KeyBackslash:  {NewTextSymbol("\\"), NewTextSymbol("|")},
		linux.KeySemicolon:  {NewTextSymbol(";"), NewTextSymbol(":")},
		linux.KeyApostrophe: {NewTextSymbol("'"), NewTextSymbol("\"")},
		linux.KeyGrave:      {NewTextSymbol("`"), NewTextSymbol("~")},
		linux.KeyComma:      {NewTextSymbol(","), NewTextSymbol("<")},
		linux.KeyDot:        {NewTextSymbol("."), NewTextSymbol(">")},
		linux.KeySlash:      {NewTextSymbol("/"), NewTextSymbol("?")},
	}
	for key, pair := range symbols {
		addEntry(mapping, key, pair[0], pair[1])
	}

	addEntry(mapping, linux.KeySpace, NewPassthroughSymbol(true), nil)

	special := []int{linux.KeyEnter, linux.KeyTab, linux.KeyEsc, linux.KeyBackspace}
	for _, key := range special {
		addEntry(mapping, key, passthrough, passthrough)
	}

	return layout
}

func buildKana86() *Layout {
	layout := NewLayout("kana-86")
	mapping := layout.mapping

	addEntry(mapping, linux.KeyQ, NewTextSymbol("た"), NewTextSymbol("タ"))
	addEntry(mapping, linux.KeyW, NewTextSymbol("て"), NewTextSymbol("テ"))
	addEntry(mapping, linux.KeyE, NewTextSymbol("い"), NewTextSymbol("イ"))
	addEntry(mapping, linux.KeyR, NewTextSymbol("す"), NewTextSymbol("ス"))
	addEntry(mapping, linux.KeyT, NewTextSymbol("か"), NewTextSymbol("カ"))
	addEntry(mapping, linux.KeyY, NewTextSymbol("ん"), NewTextSymbol("ン"))
	addEntry(mapping, linux.KeyU, NewTextSymbol("な"), NewTextSymbol("ナ"))
	addEntry(mapping, linux.KeyI, NewTextSymbol("に"), NewTextSymbol("ニ"))
	addEntry(mapping, linux.KeyO, NewTextSymbol("ら"), NewTextSymbol("ラ"))
	addEntry(mapping, linux.KeyP, NewTextSymbol("せ"), NewTextSymbol("セ"))

	addEntry(mapping, linux.KeyA, NewTextSymbol("ち"), NewTextSymbol("チ"))
	addEntry(mapping, linux.KeyS, NewTextSymbol("と"), NewTextSymbol("ト"))
	addEntry(mapping, linux.KeyD, NewTextSymbol("し"), NewTextSymbol("シ"))
	addEntry(mapping, linux.KeyF, NewTextSymbol("は"), NewTextSymbol("ハ"))
	addEntry(mapping, linux.KeyG, NewTextSymbol("き"), NewTextSymbol("キ"))
	addEntry(mapping, linux.KeyH, NewTextSymbol("く"), NewTextSymbol("ク"))
	addEntry(mapping, linux.KeyJ, NewTextSymbol("ま"), NewTextSymbol("マ"))
	addEntry(mapping, linux.KeyK, NewTextSymbol("の"), NewTextSymbol("ノ"))
	addEntry(mapping, linux.KeyL, NewTextSymbol("り"), NewTextSymbol("リ"))
	addEntry(mapping, linux.KeySemicolon, NewTextSymbol("れ"), NewTextSymbol("レ"))
	addEntry(mapping, linux.KeyApostrophe, NewTextSymbol("け"), NewTextSymbol("ケ"))

	addEntry(mapping, linux.KeyZ, NewTextSymbol("つ"), NewTextSymbol("ツ"))
	addEntry(mapping, linux.KeyX, NewTextSymbol("さ"), NewTextSymbol("サ"))
	addEntry(mapping, linux.KeyC, NewTextSymbol("そ"), NewTextSymbol("ソ"))
	addEntry(mapping, linux.KeyV, NewTextSymbol("ひ"), NewTextSymbol("ヒ"))
	addEntry(mapping, linux.KeyB, NewTextSymbol("こ"), NewTextSymbol("コ"))
	addEntry(mapping, linux.KeyN, NewTextSymbol("み"), NewTextSymbol("ミ"))
	addEntry(mapping, linux.KeyM, NewTextSymbol("も"), NewTextSymbol("モ"))
	addEntry(mapping, linux.KeyComma, NewTextSymbol("ね"), NewTextSymbol("ネ"))
	addEntry(mapping, linux.KeyDot, NewTextSymbol("る"), NewTextSymbol("ル"))
	addEntry(mapping, linux.KeySlash, NewTextSymbol("め"), NewTextSymbol("メ"))

	addEntry(mapping, linux.KeySpace, NewTextSymbol(" "), NewTextSymbol(" "))

	passthrough := NewPassthroughSymbol(true)
	special := []int{linux.KeyEnter, linux.KeyTab, linux.KeyEsc, linux.KeyBackspace}
	for _, key := range special {
		addEntry(mapping, key, passthrough, passthrough)
	}

	digits := map[int][2]*LayoutSymbol{
		linux.Key1:     {NewTextSymbol("ぬ"), NewTextSymbol("ヌ")},
		linux.Key2:     {NewTextSymbol("ふ"), NewTextSymbol("フ")},
		linux.Key3:     {NewTextSymbol("あ"), NewTextSymbol("ア")},
		linux.Key4:     {NewTextSymbol("う"), NewTextSymbol("ウ")},
		linux.Key5:     {NewTextSymbol("え"), NewTextSymbol("エ")},
		linux.Key6:     {NewTextSymbol("お"), NewTextSymbol("オ")},
		linux.Key7:     {NewTextSymbol("や"), NewTextSymbol("ヤ")},
		linux.Key8:     {NewTextSymbol("ゆ"), NewTextSymbol("ユ")},
		linux.Key9:     {NewTextSymbol("よ"), NewTextSymbol("ヨ")},
		linux.Key0:     {NewTextSymbol("わ"), NewTextSymbol("ワ")},
		linux.KeyMinus: {NewTextSymbol("ほ"), NewTextSymbol("ホ")},
		linux.KeyEqual: {NewTextSymbol("へ"), NewTextSymbol("ヘ")},
	}
	for key, pair := range digits {
		addEntry(mapping, key, pair[0], pair[1])
	}

	return layout
}

func AvailableLayouts() []string {
	names := []string{"dubeolsik", "latin", "kana-86", "sebeolsik-390"}
	sort.Strings(names)
	return names
}

func Load(name string) (*Layout, error) {
	switch name {
	case "", "dubeolsik":
		return buildDubeolsik(), nil
	case "sebeolsik-390":
		return buildSebeolsik390(), nil
	case "latin":
		return buildLatin(), nil
	case "kana-86", "kana86":
		return buildKana86(), nil
	default:
		return nil, fmt.Errorf("unknown layout: %s", name)
	}
}

func UnicodeHexKeycodes() map[rune]uint16 {
	return map[rune]uint16{
		'0': uint16(linux.Key0),
		'1': uint16(linux.Key1),
		'2': uint16(linux.Key2),
		'3': uint16(linux.Key3),
		'4': uint16(linux.Key4),
		'5': uint16(linux.Key5),
		'6': uint16(linux.Key6),
		'7': uint16(linux.Key7),
		'8': uint16(linux.Key8),
		'9': uint16(linux.Key9),
		'a': uint16(linux.KeyA),
		'b': uint16(linux.KeyB),
		'c': uint16(linux.KeyC),
		'd': uint16(linux.KeyD),
		'e': uint16(linux.KeyE),
		'f': uint16(linux.KeyF),
	}
}
