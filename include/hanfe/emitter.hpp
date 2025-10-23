#pragma once

#include <linux/input.h>
#include <linux/uinput.h>

#include <optional>
#include <string>
#include <unordered_map>

namespace hanfe {

class FallbackEmitter {
   public:
    explicit FallbackEmitter(const std::unordered_map<char, int>& hex_keys,
                             std::optional<std::string> tty_path = std::nullopt);
    ~FallbackEmitter();

    FallbackEmitter(const FallbackEmitter&) = delete;
    FallbackEmitter& operator=(const FallbackEmitter&) = delete;

    void forward_event(const input_event& event);
    void send_key_state(int keycode, bool pressed);
    void tap_key(int keycode);
    void send_backspace(int count = 1);
    void send_text(const std::string& text);
    void close();

   private:
    void emit(unsigned int type, unsigned int code, int value);
    void type_unicode(char32_t codepoint);
    void write_tty(const std::string& text);

    int uinput_fd_ = -1;
    std::unordered_map<char, int> hex_keys_;
    std::optional<int> tty_fd_;
    bool closed_ = false;
};

}  // namespace hanfe
