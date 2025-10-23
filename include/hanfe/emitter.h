#pragma once

#include <linux/input.h>
#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

struct fallback_emitter;

struct fallback_emitter* fallback_emitter_open(const int hex_keycodes[16], const char* tty_path);
void fallback_emitter_close(struct fallback_emitter* emitter);
void fallback_emitter_destroy(struct fallback_emitter* emitter);

int fallback_emitter_forward_event(struct fallback_emitter* emitter, const struct input_event* event);
int fallback_emitter_send_key_state(struct fallback_emitter* emitter, int keycode, int pressed);
int fallback_emitter_tap_key(struct fallback_emitter* emitter, int keycode);
int fallback_emitter_send_backspace(struct fallback_emitter* emitter, int count);
int fallback_emitter_send_text(struct fallback_emitter* emitter, const char* text, size_t length);

#ifdef __cplusplus
}
#endif
