#pragma once

#include <linux/input.h>

#include <memory>
#include <optional>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "hanfe/composer.hpp"
#include "hanfe/config.hpp"
#include "hanfe/emitter.h"
#include "hanfe/layout.hpp"

namespace hanfe {

struct FallbackEmitterDeleter {
    void operator()(fallback_emitter* emitter) const;
};

using FallbackEmitterPtr = std::unique_ptr<fallback_emitter, FallbackEmitterDeleter>;

class HanfeEngine {
   public:
    HanfeEngine(int device_fd, Layout layout, ToggleConfig toggle,
                FallbackEmitterPtr emitter);

    void run();

   private:
    void process_event(const input_event& event);
    void handle_modifier(const input_event& event);
    void handle_backspace(const input_event& event);
    void handle_key_release(const input_event& event);
    void handle_key_press(const input_event& event);

    void forward_key_event(const input_event& event);

    bool modifiers_active(const std::vector<int>& subset) const;
    bool shift_active() const;
    void ensure_shift_forwarded();
    void set_forwarded_modifier(int code, bool pressed);

    std::vector<int> suspend_forwarded_modifiers();
    void restore_forwarded_modifiers(const std::vector<int>& codes);

    void toggle_mode();
    void commit_text(const std::string& text);
    void commit_preedit();
    void replace_preedit(const std::string& new_text);
    void send_text(const std::string& text);

    int device_fd_;
    Layout layout_;
    ToggleConfig toggle_;
    FallbackEmitterPtr emitter_;

    HangulComposer composer_;
    InputMode mode_;
    std::string preedit_text_;
    std::unordered_set<int> toggle_keys_;

    std::unordered_map<int, bool> modifier_state_;
    std::unordered_map<int, bool> forwarded_modifiers_;
    std::unordered_set<int> forwarded_keys_;
};

}  // namespace hanfe
