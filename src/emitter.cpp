#include "hanfe/emitter.hpp"

#include <fcntl.h>
#include <sys/ioctl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

#include <cerrno>
#include <cstdio>
#include <cstring>
#include <cstdint>
#include <iomanip>
#include <sstream>
#include <stdexcept>

#include "hanfe/util.hpp"

namespace hanfe {
namespace {

constexpr int kMaxKeyCode = KEY_MAX;

void set_keybits(int fd) {
    if (ioctl(fd, UI_SET_EVBIT, EV_SYN) < 0 || ioctl(fd, UI_SET_EVBIT, EV_KEY) < 0) {
        throw std::runtime_error("Failed to configure uinput event bits: " +
                                 std::string(strerror(errno)));
    }
    for (int code = 0; code <= kMaxKeyCode; ++code) {
        ioctl(fd, UI_SET_KEYBIT, code);
    }
}

}  // namespace

FallbackEmitter::FallbackEmitter(const std::unordered_map<char, int>& hex_keys,
                                 std::optional<std::string> tty_path)
    : hex_keys_(hex_keys) {
    uinput_fd_ = ::open("/dev/uinput", O_WRONLY | O_NONBLOCK | O_CLOEXEC);
    if (uinput_fd_ < 0) {
        throw std::runtime_error("Failed to open /dev/uinput: " + std::string(strerror(errno)));
    }

    set_keybits(uinput_fd_);

    struct uinput_user_dev setup;
    std::memset(&setup, 0, sizeof(setup));
    std::snprintf(setup.name, UINPUT_MAX_NAME_SIZE, "hanfe-fallback");
    setup.id.bustype = BUS_USB;
    setup.id.vendor = 0x1;
    setup.id.product = 0x1;
    setup.id.version = 1;

    if (write(uinput_fd_, &setup, sizeof(setup)) < 0) {
        throw std::runtime_error("Failed to configure uinput device: " +
                                 std::string(strerror(errno)));
    }
    if (ioctl(uinput_fd_, UI_DEV_CREATE) < 0) {
        throw std::runtime_error("Failed to create uinput device: " +
                                 std::string(strerror(errno)));
    }

    if (tty_path) {
        int fd = ::open(tty_path->c_str(), O_WRONLY | O_NOCTTY | O_CLOEXEC);
        if (fd < 0) {
            throw std::runtime_error("Failed to open TTY '" + *tty_path + "': " +
                                     std::string(strerror(errno)));
        }
        tty_fd_ = fd;
    }
}

FallbackEmitter::~FallbackEmitter() { close(); }

void FallbackEmitter::close() {
    if (closed_) {
        return;
    }
    closed_ = true;
    if (uinput_fd_ >= 0) {
        ioctl(uinput_fd_, UI_DEV_DESTROY);
        ::close(uinput_fd_);
        uinput_fd_ = -1;
    }
    if (tty_fd_) {
        ::close(*tty_fd_);
        tty_fd_.reset();
    }
}

void FallbackEmitter::emit(unsigned int type, unsigned int code, int value) {
    if (uinput_fd_ < 0) {
        return;
    }
    input_event ev{};
    ev.type = type;
    ev.code = static_cast<unsigned short>(code);
    ev.value = value;
    if (write(uinput_fd_, &ev, sizeof(ev)) < 0) {
        throw std::runtime_error("Failed to write to uinput: " + std::string(strerror(errno)));
    }
    input_event syn{};
    syn.type = EV_SYN;
    syn.code = SYN_REPORT;
    syn.value = 0;
    if (write(uinput_fd_, &syn, sizeof(syn)) < 0) {
        throw std::runtime_error("Failed to sync uinput: " + std::string(strerror(errno)));
    }
}

void FallbackEmitter::forward_event(const input_event& event) {
    emit(event.type, event.code, event.value);
}

void FallbackEmitter::send_key_state(int keycode, bool pressed) {
    emit(EV_KEY, keycode, pressed ? 1 : 0);
}

void FallbackEmitter::tap_key(int keycode) {
    send_key_state(keycode, true);
    send_key_state(keycode, false);
}

void FallbackEmitter::send_backspace(int count) {
    for (int i = 0; i < count; ++i) {
        tap_key(KEY_BACKSPACE);
        write_tty("\b");
    }
}

void FallbackEmitter::send_text(const std::string& text) {
    if (text.empty()) {
        return;
    }
    std::u32string codepoints = utf8_to_u32(text);
    for (char32_t cp : codepoints) {
        type_unicode(cp);
        write_tty(utf8_from_char32(cp));
    }
}

// Helper: convert codepoint -> UTF-8
static std::string codepoint_to_utf8(char32_t cp) {
    std::string out;
    if (cp <= 0x7F) {
        out.push_back(static_cast<char>(cp));
    } else if (cp <= 0x7FF) {
        out.push_back(static_cast<char>(0xC0 | ((cp >> 6) & 0x1F)));
        out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
    } else if (cp <= 0xFFFF) {
        out.push_back(static_cast<char>(0xE0 | ((cp >> 12) & 0x0F)));
        out.push_back(static_cast<char>(0x80 | ((cp >> 6) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
    } else {
        out.push_back(static_cast<char>(0xF0 | ((cp >> 18) & 0x07)));
        out.push_back(static_cast<char>(0x80 | ((cp >> 12) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | ((cp >> 6) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | (cp & 0x3F)));
    }
    return out;
}

void FallbackEmitter::type_unicode(char32_t codepoint) {
    // keep original modifier+U sequence so GUI fallback stays intact
    send_key_state(KEY_LEFTCTRL, true);
    send_key_state(KEY_LEFTSHIFT, true);
    tap_key(KEY_U);
    send_key_state(KEY_LEFTSHIFT, false);
    send_key_state(KEY_LEFTCTRL, false);

    // write the real UTF-8 to TTY for visibility/logging
    write_tty(codepoint_to_utf8(codepoint));

    // *** If a TTY is open, DO NOT type hex digits (they would duplicate / conflict) ***
    if (tty_fd_) {
        return;
    }

    // Otherwise (no tty), fall back to typing hex digits as before
    std::ostringstream oss;
    oss << std::hex << std::nouppercase << static_cast<uint32_t>(codepoint);
    std::string hex = oss.str();

    for (char ch : hex) {
        auto it = hex_keys_.find(ch);
        if (it == hex_keys_.end()) continue;
        tap_key(it->second);
    }
}

void FallbackEmitter::write_tty(const std::string& text) {
    if (!tty_fd_ || text.empty()) {
        return;
    }
    ssize_t ignored = ::write(*tty_fd_, text.data(), text.size());
    (void)ignored;
}

}  // namespace hanfe
