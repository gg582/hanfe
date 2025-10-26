package emitter

import "github.com/gg582/hanfe/internal/util"

// Output represents the operations required by the engine to emit characters,
// update preedit text, and forward raw input events. It is satisfied by
// FallbackEmitter and enables tests to substitute lightweight fakes.
type Output interface {
	Close() error
	ForwardEvent(*util.InputEvent) error
	SendKeyState(code uint16, pressed bool) error
	TapKey(code uint16) error
	SendBackspace(count int) error
	SendText(text string) error
	SupportsPreedit() bool
}

var _ Output = (*FallbackEmitter)(nil)
