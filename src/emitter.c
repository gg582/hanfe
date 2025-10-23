#include "hanfe/emitter.h"

#include <errno.h>
#include <fcntl.h>
#include <linux/input-event-codes.h>
#include <linux/uinput.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/ioctl.h>
#include <sys/types.h>
#include <termios.h>
#include <unistd.h>

struct fallback_emitter {
    int uinput_fd;
    int tty_fd;
    int closed;
    int hex_keycodes[16];
};

static void fill_hex_keycodes(struct fallback_emitter* emitter, const int hex_keycodes[16]) {
    for (int i = 0; i < 16; ++i) {
        emitter->hex_keycodes[i] = (hex_keycodes ? hex_keycodes[i] : -1);
    }
}

static int open_uinput(void) {
    return open("/dev/uinput", O_WRONLY | O_NONBLOCK | O_CLOEXEC);
}

static int configure_uinput(int fd) {
    if (fd < 0) {
        return -1;
    }

    if (ioctl(fd, UI_SET_EVBIT, EV_SYN) < 0 || ioctl(fd, UI_SET_EVBIT, EV_KEY) < 0) {
        return -1;
    }

    for (int code = 0; code <= KEY_MAX; ++code) {
        ioctl(fd, UI_SET_KEYBIT, code);
    }

    struct uinput_user_dev setup;
    memset(&setup, 0, sizeof(setup));
    snprintf(setup.name, UINPUT_MAX_NAME_SIZE, "hanfe-fallback");
    setup.id.bustype = BUS_USB;
    setup.id.vendor = 0x1;
    setup.id.product = 0x1;
    setup.id.version = 1;

    if (write(fd, &setup, sizeof(setup)) < 0) {
        return -1;
    }
    if (ioctl(fd, UI_DEV_CREATE) < 0) {
        return -1;
    }
    return 0;
}

static int open_tty(const char* tty_path) {
    if (!tty_path) {
        return -1;
    }
    return open(tty_path, O_WRONLY | O_NOCTTY | O_CLOEXEC);
}

struct fallback_emitter* fallback_emitter_open(const int hex_keycodes[16], const char* tty_path) {
    struct fallback_emitter* emitter = (struct fallback_emitter*)calloc(1, sizeof(*emitter));
    if (!emitter) {
        return NULL;
    }
    emitter->uinput_fd = -1;
    emitter->tty_fd = -1;
    emitter->closed = 0;
    fill_hex_keycodes(emitter, hex_keycodes);

    int fd = open_uinput();
    if (fd < 0) {
        int saved = errno;
        free(emitter);
        errno = saved;
        return NULL;
    }
    if (configure_uinput(fd) < 0) {
        int saved = errno;
        close(fd);
        free(emitter);
        errno = saved;
        return NULL;
    }
    emitter->uinput_fd = fd;

    if (tty_path) {
        int tty_fd = open_tty(tty_path);
        if (tty_fd < 0) {
            int saved = errno;
            ioctl(emitter->uinput_fd, UI_DEV_DESTROY);
            close(emitter->uinput_fd);
            free(emitter);
            errno = saved;
            return NULL;
        }
        emitter->tty_fd = tty_fd;
    }

    return emitter;
}

void fallback_emitter_close(struct fallback_emitter* emitter) {
    if (!emitter || emitter->closed) {
        return;
    }
    emitter->closed = 1;
    if (emitter->uinput_fd >= 0) {
        ioctl(emitter->uinput_fd, UI_DEV_DESTROY);
        close(emitter->uinput_fd);
        emitter->uinput_fd = -1;
    }
    if (emitter->tty_fd >= 0) {
        close(emitter->tty_fd);
        emitter->tty_fd = -1;
    }
}

void fallback_emitter_destroy(struct fallback_emitter* emitter) {
    if (!emitter) {
        return;
    }
    fallback_emitter_close(emitter);
    free(emitter);
}

static int write_event(int fd, const struct input_event* ev) {
    ssize_t written = write(fd, ev, sizeof(*ev));
    if (written < 0) {
        return -1;
    }
    if ((size_t)written != sizeof(*ev)) {
        errno = EIO;
        return -1;
    }
    return 0;
}

static int emit_sync(struct fallback_emitter* emitter) {
    if (!emitter || emitter->uinput_fd < 0) {
        return 0;
    }
    struct input_event syn;
    memset(&syn, 0, sizeof(syn));
    syn.type = EV_SYN;
    syn.code = SYN_REPORT;
    syn.value = 0;
    return write_event(emitter->uinput_fd, &syn);
}

static int emit_key_event(struct fallback_emitter* emitter, unsigned int type, unsigned int code, int value) {
    if (!emitter || emitter->uinput_fd < 0) {
        return 0;
    }
    struct input_event ev;
    memset(&ev, 0, sizeof(ev));
    ev.type = type;
    ev.code = (unsigned short)code;
    ev.value = value;
    if (write_event(emitter->uinput_fd, &ev) < 0) {
        return -1;
    }
    return emit_sync(emitter);
}

int fallback_emitter_forward_event(struct fallback_emitter* emitter, const struct input_event* event) {
    if (!emitter || emitter->uinput_fd < 0 || !event) {
        return 0;
    }
    if (write_event(emitter->uinput_fd, event) < 0) {
        return -1;
    }
    return emit_sync(emitter);
}

int fallback_emitter_send_key_state(struct fallback_emitter* emitter, int keycode, int pressed) {
    return emit_key_event(emitter, EV_KEY, (unsigned int)keycode, pressed ? 1 : 0);
}

int fallback_emitter_tap_key(struct fallback_emitter* emitter, int keycode) {
    if (fallback_emitter_send_key_state(emitter, keycode, 1) < 0) {
        return -1;
    }
    return fallback_emitter_send_key_state(emitter, keycode, 0);
}

static int tty_push_byte(struct fallback_emitter* emitter, unsigned char byte) {
    if (!emitter || emitter->tty_fd < 0) {
        return 0;
    }
    if (ioctl(emitter->tty_fd, TIOCSTI, &byte) < 0) {
        ssize_t written = write(emitter->tty_fd, &byte, 1);
        if (written < 0) {
            return -1;
        }
        if (written != 1) {
            errno = EIO;
            return -1;
        }
    }
    return 0;
}

static int tty_write_bytes(struct fallback_emitter* emitter, const char* data, size_t len) {
    if (!emitter || emitter->tty_fd < 0 || !data || len == 0) {
        return 0;
    }
    for (size_t i = 0; i < len; ++i) {
        if (tty_push_byte(emitter, (unsigned char)data[i]) < 0) {
            return -1;
        }
    }
    return 0;
}

int fallback_emitter_send_backspace(struct fallback_emitter* emitter, int count) {
    if (count <= 0) {
        return 0;
    }
    for (int i = 0; i < count; ++i) {
        if (fallback_emitter_tap_key(emitter, KEY_BACKSPACE) < 0) {
            return -1;
        }
        if (tty_push_byte(emitter, '\b') < 0) {
            return -1;
        }
    }
    return 0;
}

static size_t decode_utf8(const char* data, size_t length, size_t offset, uint32_t* out_cp) {
    if (!data || offset >= length) {
        return 0;
    }
    unsigned char first = (unsigned char)data[offset];
    if (first < 0x80) {
        *out_cp = first;
        return 1;
    }
    if ((first & 0xE0) == 0xC0 && offset + 1 < length) {
        unsigned char b1 = (unsigned char)data[offset + 1];
        if ((b1 & 0xC0) != 0x80) {
            return 0;
        }
        *out_cp = ((uint32_t)(first & 0x1F) << 6) | (uint32_t)(b1 & 0x3F);
        if (*out_cp < 0x80) {
            return 0;
        }
        return 2;
    }
    if ((first & 0xF0) == 0xE0 && offset + 2 < length) {
        unsigned char b1 = (unsigned char)data[offset + 1];
        unsigned char b2 = (unsigned char)data[offset + 2];
        if ((b1 & 0xC0) != 0x80 || (b2 & 0xC0) != 0x80) {
            return 0;
        }
        *out_cp = ((uint32_t)(first & 0x0F) << 12) | ((uint32_t)(b1 & 0x3F) << 6) |
                  (uint32_t)(b2 & 0x3F);
        if (*out_cp < 0x800) {
            return 0;
        }
        return 3;
    }
    if ((first & 0xF8) == 0xF0 && offset + 3 < length) {
        unsigned char b1 = (unsigned char)data[offset + 1];
        unsigned char b2 = (unsigned char)data[offset + 2];
        unsigned char b3 = (unsigned char)data[offset + 3];
        if ((b1 & 0xC0) != 0x80 || (b2 & 0xC0) != 0x80 || (b3 & 0xC0) != 0x80) {
            return 0;
        }
        *out_cp = ((uint32_t)(first & 0x07) << 18) | ((uint32_t)(b1 & 0x3F) << 12) |
                  ((uint32_t)(b2 & 0x3F) << 6) | (uint32_t)(b3 & 0x3F);
        if (*out_cp < 0x10000 || *out_cp > 0x10FFFF) {
            return 0;
        }
        return 4;
    }
    return 0;
}

static int hex_index(char ch) {
    if (ch >= '0' && ch <= '9') {
        return ch - '0';
    }
    if (ch >= 'a' && ch <= 'f') {
        return 10 + (ch - 'a');
    }
    if (ch >= 'A' && ch <= 'F') {
        return 10 + (ch - 'A');
    }
    return -1;
}

static int codepoint_to_hex(char buffer[9], uint32_t codepoint) {
    if (!buffer) {
        return 0;
    }
    if (codepoint == 0) {
        buffer[0] = '0';
        buffer[1] = '\0';
        return 1;
    }
    int pos = 0;
    char temp[8];
    while (codepoint > 0 && pos < 8) {
        uint32_t nibble = codepoint & 0xF;
        temp[pos++] = (nibble < 10) ? (char)('0' + nibble) : (char)('a' + (nibble - 10));
        codepoint >>= 4;
    }
    for (int i = 0; i < pos; ++i) {
        buffer[i] = temp[pos - 1 - i];
    }
    buffer[pos] = '\0';
    return pos;
}

static int type_unicode(struct fallback_emitter* emitter, uint32_t codepoint) {
    if (!emitter || emitter->uinput_fd < 0) {
        return 0;
    }

    if (fallback_emitter_send_key_state(emitter, KEY_LEFTCTRL, 1) < 0) {
        return -1;
    }
    if (fallback_emitter_send_key_state(emitter, KEY_LEFTSHIFT, 1) < 0) {
        return -1;
    }
    if (fallback_emitter_tap_key(emitter, KEY_U) < 0) {
        return -1;
    }
    if (fallback_emitter_send_key_state(emitter, KEY_LEFTSHIFT, 0) < 0) {
        return -1;
    }
    if (fallback_emitter_send_key_state(emitter, KEY_LEFTCTRL, 0) < 0) {
        return -1;
    }

    char hex[9];
    int hex_len = codepoint_to_hex(hex, codepoint);
    for (int i = 0; i < hex_len; ++i) {
        int idx = hex_index(hex[i]);
        if (idx < 0) {
            continue;
        }
        int keycode = emitter->hex_keycodes[idx];
        if (keycode < 0) {
            continue;
        }
        if (fallback_emitter_tap_key(emitter, keycode) < 0) {
            return -1;
        }
    }

    if (fallback_emitter_send_key_state(emitter, KEY_LEFTCTRL, 1) < 0) {
        return -1;
    }
    if (fallback_emitter_send_key_state(emitter, KEY_LEFTSHIFT, 1) < 0) {
        return -1;
    }
    if (fallback_emitter_tap_key(emitter, KEY_ENTER) < 0) {
        return -1;
    }
    if (fallback_emitter_send_key_state(emitter, KEY_LEFTSHIFT, 0) < 0) {
        return -1;
    }
    if (fallback_emitter_send_key_state(emitter, KEY_LEFTCTRL, 0) < 0) {
        return -1;
    }

    return 0;
}

int fallback_emitter_send_text(struct fallback_emitter* emitter, const char* text, size_t length) {
    if (!emitter || !text || length == 0) {
        return 0;
    }
    size_t offset = 0;
    while (offset < length) {
        uint32_t codepoint = 0;
        size_t consumed = decode_utf8(text, length, offset, &codepoint);
        if (consumed == 0) {
            errno = EINVAL;
            return -1;
        }
        if (tty_write_bytes(emitter, text + offset, consumed) < 0) {
            return -1;
        }
        if (type_unicode(emitter, codepoint) < 0) {
            return -1;
        }
        offset += consumed;
    }
    return 0;
}
