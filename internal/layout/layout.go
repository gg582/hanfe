package layout

import (
	"fmt"
	"sort"

	"github.com/gg582/hanfe/internal/hangul"
	"github.com/gg582/hanfe/internal/linux"
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

type Category string

const (
	CategoryHangul Category = "hangul"
	CategoryKana   Category = "kana"
)

type Layout struct {
	name     string
	category Category
	mapping  map[uint16]LayoutEntry
}

func (l Layout) Name() string { return l.name }

func (l Layout) Category() Category { return l.category }

func (l Layout) Translate(code uint16, shift bool) *LayoutSymbol {
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

func makeJamoSymbol(value rune, role hangul.JamoRole) *LayoutSymbol {
	return &LayoutSymbol{Kind: SymbolJamo, Jamo: value, Role: role}
}

func makeJamo(value rune) *LayoutSymbol { return makeJamoSymbol(value, hangul.RoleAuto) }

func makePassthroughSymbol(commitBefore bool) *LayoutSymbol {
	return &LayoutSymbol{Kind: SymbolPassthrough, CommitBefore: commitBefore}
}

func addEntry(mapping map[uint16]LayoutEntry, key int, normal *LayoutSymbol, shifted *LayoutSymbol) {
	mapping[uint16(key)] = LayoutEntry{Normal: normal, Shifted: shifted}
}

func buildDubeolsik() Layout {
	mapping := make(map[uint16]LayoutEntry)
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

	return Layout{name: "dubeolsik", category: CategoryHangul, mapping: mapping}
}

func buildSebeolsik390() Layout {
	mapping := make(map[uint16]LayoutEntry)
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

	return Layout{name: "sebeolsik-390", category: CategoryHangul, mapping: mapping}
}

func makeTextSymbol(value string, commitBefore bool) *LayoutSymbol {
	return &LayoutSymbol{Kind: SymbolText, Text: value, CommitBefore: commitBefore}
}

func hiraganaToKatakana(r rune) rune {
	// Hiragana and Katakana blocks are offset by a constant value.
	const hiraganaStart = rune(0x3041)
	const hiraganaEnd = rune(0x3096)
	const katakanaStart = rune(0x30A1)
	offset := katakanaStart - hiraganaStart
	if r >= hiraganaStart && r <= hiraganaEnd {
		return r + offset
	}
	return r
}

func hiraganaPair(normal string) (string, string) {
	if normal == "" {
		return "", ""
	}
	runes := []rune(normal)
	shifted := make([]rune, len(runes))
	for i, r := range runes {
		shifted[i] = hiraganaToKatakana(r)
	}
	return normal, string(shifted)
}

func buildKana86() Layout {
	mapping := make(map[uint16]LayoutEntry)

	addKana := func(key int, normal string) {
		n, s := hiraganaPair(normal)
		addEntry(mapping, key, makeTextSymbol(n, false), makeTextSymbol(s, false))
	}

	addKana(linux.Key1, "ぬ")
	addKana(linux.Key2, "ふ")
	addKana(linux.Key3, "あ")
	addKana(linux.Key4, "う")
	addKana(linux.Key5, "え")
	addKana(linux.Key6, "お")
	addKana(linux.Key7, "や")
	addKana(linux.Key8, "ゆ")
	addKana(linux.Key9, "よ")
	addKana(linux.Key0, "わ")
	addKana(linux.KeyMinus, "ほ")
	addKana(linux.KeyEqual, "へ")

	addKana(linux.KeyQ, "た")
	addKana(linux.KeyW, "て")
	addKana(linux.KeyE, "い")
	addKana(linux.KeyR, "す")
	addKana(linux.KeyT, "か")
	addKana(linux.KeyY, "ん")
	addKana(linux.KeyU, "な")
	addKana(linux.KeyI, "に")
	addKana(linux.KeyO, "ら")
	addKana(linux.KeyP, "せ")
	addKana(linux.KeyLeftBrace, "゛")
	addKana(linux.KeyRightBrace, "゜")

	addKana(linux.KeyA, "ち")
	addKana(linux.KeyS, "と")
	addKana(linux.KeyD, "し")
	addKana(linux.KeyF, "は")
	addKana(linux.KeyG, "き")
	addKana(linux.KeyH, "く")
	addKana(linux.KeyJ, "ま")
	addKana(linux.KeyK, "の")
	addKana(linux.KeyL, "り")
	addKana(linux.KeySemicolon, "れ")
	addKana(linux.KeyApostrophe, "け")
	addKana(linux.KeyBackslash, "む")

	addKana(linux.KeyZ, "つ")
	addKana(linux.KeyX, "さ")
	addKana(linux.KeyC, "そ")
	addKana(linux.KeyV, "ひ")
	addKana(linux.KeyB, "こ")
	addKana(linux.KeyN, "み")
	addKana(linux.KeyM, "も")
	addKana(linux.KeyComma, "ね")
	addKana(linux.KeyDot, "る")
	addKana(linux.KeySlash, "め")

	space := makePassthroughSymbol(true)
	addEntry(mapping, linux.KeySpace, space, nil)

	passthroughKeys := []int{linux.KeyTab, linux.KeyEnter, linux.KeyEsc, linux.KeyBackspace}
	for _, key := range passthroughKeys {
		addEntry(mapping, key, space, nil)
	}

	return Layout{name: "kana86", category: CategoryKana, mapping: mapping}
}

func AvailableLayouts() []string {
	names := []string{"dubeolsik", "kana86", "sebeolsik-390"}
	sort.Strings(names)
	return names
}

func Load(name string) (Layout, error) {
	switch name {
	case "", "dubeolsik":
		return buildDubeolsik(), nil
	case "sebeolsik-390":
		return buildSebeolsik390(), nil
	case "kana86":
		return buildKana86(), nil
	default:
		return Layout{}, fmt.Errorf("unknown layout: %s", name)
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
