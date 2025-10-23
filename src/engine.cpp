#include "hanfe/engine.hpp"

#include <linux/input-event-codes.h>

#include <algorithm>
#include <cerrno>
#include <cstring>
#include <stdexcept>
#include <sys/ioctl.h>
#include <unistd.h>

#include "hanfe/util.hpp"

namespace hanfe {
namespace {

const std::vector<int> SHIFT_KEYS = {KEY_LEFTSHIFT, KEY_RIGHTSHIFT};
const std::vector<int> CTRL_KEYS = {KEY_LEFTCTRL, KEY_RIGHTCTRL};
const std::vector<int> ALT_KEYS = {KEY_LEFTALT, KEY_RIGHTALT};
const std::vector<int> META_KEYS = {KEY_LEFTMETA, KEY_RIGHTMETA};

std::vector<int> combine(const std::vector<int>& a, const std::vector<int>& b) {
    std::vector<int> result = a;
    result.insert(result.end(), b.begin(), b.end());
    return result;
}

const std::vector<int> MODIFIER_KEYS = combine(combine(combine(SHIFT_KEYS, CTRL_KEYS), ALT_KEYS), META_KEYS);
const std::vector<int> ALWAYS_FORWARD = combine(combine(CTRL_KEYS, ALT_KEYS), META_KEYS);

bool is_key_press(const input_event& event) { return event.value == 1 || event.value == 2; }

bool is_key_release(const input_event& event) { return event.value == 0; }

}  // namespace

HanfeEngine::HanfeEngine(int device_fd, Layout layout, ToggleConfig toggle,
                         std::unique_ptr<FallbackEmitter> emitter)
    : device_fd_(device_fd),
      layout_(std::move(layout)),
      toggle_(std::move(toggle)),
      emitter_(std::move(emitter)),
      mode_(toggle_.default_mode),
      preedit_text_("") {
    if (device_fd_ < 0) {
        throw std::runtime_error("Invalid device descriptor");
    }
    toggle_keys_.insert(toggle_.toggle_keys.begin(), toggle_.toggle_keys.end());
    for (int code : MODIFIER_KEYS) {
        modifier_state_[code] = false;
        forwarded_modifiers_[code] = false;
    }
}

void HanfeEngine::run() {
    if (ioctl(device_fd_, EVIOCGRAB, 1) < 0) {
        throw std::runtime_error("Failed to grab device: " + std::string(strerror(errno)));
    }
    try {
        input_event event{};
        while (true) {
            ssize_t n = ::read(device_fd_, &event, sizeof(event));
            if (n < 0) {
                if (errno == EAGAIN || errno == EINTR) {
                    continue;
                }
                throw std::runtime_error("Failed to read input event: " +
                                         std::string(strerror(errno)));
            }
            if (n == 0) {
                break;
            }
            if (n != sizeof(event)) {
                continue;
            }
            process_event(event);
        }
    } catch (...) {
        ioctl(device_fd_, EVIOCGRAB, 0);
        emitter_->close();
        throw;
    }
    ioctl(device_fd_, EVIOCGRAB, 0);
    emitter_->close();
}

void HanfeEngine::process_event(const input_event& event) {
    if (event.type != EV_KEY) {
        if (mode_ == InputMode::Latin) {
            emitter_->forward_event(event);
        }
        return;
    }

    int code = event.code;
    if (toggle_keys_.count(code) > 0) {
        if (event.value == 1) {
            toggle_mode();
        }
        return;
    }

    if (std::find(MODIFIER_KEYS.begin(), MODIFIER_KEYS.end(), code) != MODIFIER_KEYS.end()) {
        handle_modifier(event);
        return;
    }

    if (mode_ == InputMode::Latin) {
        forward_key_event(event);
        return;
    }

    if (code == KEY_BACKSPACE) {
        handle_backspace(event);
        return;
    }

    if (is_key_release(event)) {
        handle_key_release(event);
        return;
    }

    handle_key_press(event);
}

void HanfeEngine::handle_modifier(const input_event& event) {
    int code = event.code;
    bool press = is_key_press(event);
    bool release = is_key_release(event);

    if (press) {
        modifier_state_[code] = true;
    } else if (release) {
        modifier_state_[code] = false;
    }

    if (mode_ == InputMode::Latin ||
        std::find(ALWAYS_FORWARD.begin(), ALWAYS_FORWARD.end(), code) != ALWAYS_FORWARD.end()) {
        forward_key_event(event);
        forwarded_modifiers_[code] = press && !release;
        return;
    }

    if (release && forwarded_modifiers_[code]) {
        set_forwarded_modifier(code, false);
    }
}

void HanfeEngine::handle_backspace(const input_event& event) {
    if (mode_ == InputMode::Latin) {
        forward_key_event(event);
        return;
    }
    if (is_key_release(event)) {
        if (forwarded_keys_.count(KEY_BACKSPACE)) {
            forward_key_event(event);
        }
        return;
    }

    auto new_preedit = composer_.backspace();
    if (new_preedit) {
        replace_preedit(*new_preedit);
        return;
    }
    commit_preedit();
    forward_key_event(event);
}

void HanfeEngine::handle_key_release(const input_event& event) {
    int code = event.code;
    if (forwarded_keys_.count(code) > 0) {
        forward_key_event(event);
    }
}

void HanfeEngine::handle_key_press(const input_event& event) {
    int code = event.code;
    if (modifiers_active(ALWAYS_FORWARD)) {
        commit_preedit();
        ensure_shift_forwarded();
        forward_key_event(event);
        return;
    }

    const LayoutSymbol* symbol = layout_.translate(code, shift_active());
    if (!symbol) {
        commit_preedit();
        ensure_shift_forwarded();
        forward_key_event(event);
        return;
    }

    switch (symbol->kind) {
        case SymbolKind::Passthrough: {
            if (symbol->commit_before) {
                commit_preedit();
            }
            ensure_shift_forwarded();
            forward_key_event(event);
            return;
        }
        case SymbolKind::Text: {
            if (symbol->commit_before) {
                commit_preedit();
            }
            send_text(symbol->text);
            return;
        }
        case SymbolKind::Jamo: {
            CompositionResult result = composer_.feed(symbol->jamo, symbol->role);
            if (!result.commit.empty()) {
                commit_text(result.commit);
            }
            if (result.preedit != preedit_text_) {
                replace_preedit(result.preedit);
            }
            return;
        }
    }
}

void HanfeEngine::forward_key_event(const input_event& event) {
    emitter_->forward_event(event);
    if (is_key_press(event)) {
        forwarded_keys_.insert(event.code);
    } else if (is_key_release(event)) {
        forwarded_keys_.erase(event.code);
    }
}

bool HanfeEngine::modifiers_active(const std::vector<int>& subset) const {
    for (int code : subset) {
        auto it = modifier_state_.find(code);
        if (it != modifier_state_.end() && it->second) {
            return true;
        }
    }
    return false;
}

bool HanfeEngine::shift_active() const { return modifiers_active(SHIFT_KEYS); }

void HanfeEngine::ensure_shift_forwarded() {
    for (int code : SHIFT_KEYS) {
        if (modifier_state_.at(code) && !forwarded_modifiers_.at(code)) {
            set_forwarded_modifier(code, true);
        }
    }
}

void HanfeEngine::set_forwarded_modifier(int code, bool pressed) {
    auto it = forwarded_modifiers_.find(code);
    if (it != forwarded_modifiers_.end() && it->second == pressed) {
        return;
    }
    emitter_->send_key_state(code, pressed);
    forwarded_modifiers_[code] = pressed;
}

std::vector<int> HanfeEngine::suspend_forwarded_modifiers() {
    std::vector<int> suspended;
    for (auto& [code, forwarded] : forwarded_modifiers_) {
        if (forwarded) {
            set_forwarded_modifier(code, false);
            suspended.push_back(code);
        }
    }
    return suspended;
}

void HanfeEngine::restore_forwarded_modifiers(const std::vector<int>& codes) {
    for (int code : codes) {
        if (modifier_state_.at(code)) {
            set_forwarded_modifier(code, true);
        }
    }
}

void HanfeEngine::toggle_mode() {
    commit_preedit();
    mode_ = (mode_ == InputMode::Hangul) ? InputMode::Latin : InputMode::Hangul;
}

void HanfeEngine::commit_text(const std::string& text) {
    if (text.empty()) {
        return;
    }
    replace_preedit("");
    send_text(text);
}

void HanfeEngine::commit_preedit() {
    std::string commit = composer_.flush();
    if (commit.empty() && preedit_text_.empty()) {
        return;
    }
    replace_preedit("");
    if (!commit.empty()) {
        send_text(commit);
    }
}

void HanfeEngine::replace_preedit(const std::string& new_text) {
    if (new_text == preedit_text_) {
        return;
    }
    auto suspended = suspend_forwarded_modifiers();
    if (!preedit_text_.empty()) {
        size_t old_count = utf8_to_u32(preedit_text_).size();
        if (old_count > 0) {
            emitter_->send_backspace(static_cast<int>(old_count));
        }
    }
    if (!new_text.empty()) {
        emitter_->send_text(new_text);
    }
    preedit_text_ = new_text;
    restore_forwarded_modifiers(suspended);
}

void HanfeEngine::send_text(const std::string& text) {
    if (text.empty()) {
        return;
    }
    auto suspended = suspend_forwarded_modifiers();
    emitter_->send_text(text);
    restore_forwarded_modifiers(suspended);
}

}  // namespace hanfe
