#include "hanfe/config.hpp"

#include <linux/input-event-codes.h>

#include <algorithm>
#include <cctype>
#include <fstream>
#include <unordered_map>

#include "hanfe/util.hpp"

namespace hanfe {
namespace {

const std::unordered_map<std::string, int>& keycode_table() {
    static const std::unordered_map<std::string, int> table = [] {
        std::unordered_map<std::string, int> map;
        for (char ch = 'A'; ch <= 'Z'; ++ch) {
            std::string name = "KEY_" + std::string(1, ch);
            map.emplace(name, KEY_A + (ch - 'A'));
        }
        for (char ch = '0'; ch <= '9'; ++ch) {
            std::string name = "KEY_" + std::string(1, ch);
            map.emplace(name, KEY_0 + (ch - '0'));
        }
        map.emplace("KEY_MINUS", KEY_MINUS);
        map.emplace("KEY_EQUAL", KEY_EQUAL);
        map.emplace("KEY_LEFTBRACE", KEY_LEFTBRACE);
        map.emplace("KEY_RIGHTBRACE", KEY_RIGHTBRACE);
        map.emplace("KEY_BACKSLASH", KEY_BACKSLASH);
        map.emplace("KEY_SEMICOLON", KEY_SEMICOLON);
        map.emplace("KEY_APOSTROPHE", KEY_APOSTROPHE);
        map.emplace("KEY_GRAVE", KEY_GRAVE);
        map.emplace("KEY_COMMA", KEY_COMMA);
        map.emplace("KEY_DOT", KEY_DOT);
        map.emplace("KEY_SLASH", KEY_SLASH);
        map.emplace("KEY_SPACE", KEY_SPACE);
        map.emplace("KEY_TAB", KEY_TAB);
        map.emplace("KEY_ENTER", KEY_ENTER);
        map.emplace("KEY_ESC", KEY_ESC);
        map.emplace("KEY_BACKSPACE", KEY_BACKSPACE);
        map.emplace("KEY_LEFTSHIFT", KEY_LEFTSHIFT);
        map.emplace("KEY_RIGHTSHIFT", KEY_RIGHTSHIFT);
        map.emplace("KEY_LEFTCTRL", KEY_LEFTCTRL);
        map.emplace("KEY_RIGHTCTRL", KEY_RIGHTCTRL);
        map.emplace("KEY_LEFTALT", KEY_LEFTALT);
        map.emplace("KEY_RIGHTALT", KEY_RIGHTALT);
        map.emplace("KEY_LEFTMETA", KEY_LEFTMETA);
        map.emplace("KEY_RIGHTMETA", KEY_RIGHTMETA);
        map.emplace("KEY_HANGUL", KEY_HANGEUL);
        map.emplace("KEY_HANJA", KEY_HANJA);
        map.emplace("KEY_CAPSLOCK", KEY_CAPSLOCK);
        map.emplace("KEY_F1", KEY_F1);
        map.emplace("KEY_F2", KEY_F2);
        map.emplace("KEY_F3", KEY_F3);
        map.emplace("KEY_F4", KEY_F4);
        map.emplace("KEY_F5", KEY_F5);
        map.emplace("KEY_F6", KEY_F6);
        map.emplace("KEY_F7", KEY_F7);
        map.emplace("KEY_F8", KEY_F8);
        map.emplace("KEY_F9", KEY_F9);
        map.emplace("KEY_F10", KEY_F10);
        map.emplace("KEY_F11", KEY_F11);
        map.emplace("KEY_F12", KEY_F12);
        return map;
    }();
    return table;
}

int parse_keycode(std::string name, const std::string& source) {
    if (name.empty()) {
        throw ConfigError("Empty key name in " + source);
    }
    std::string upper;
    upper.reserve(name.size());
    for (char ch : name) {
        upper.push_back(static_cast<char>(std::toupper(static_cast<unsigned char>(ch))));
    }
    if (upper == "ALT_R") {
        upper = "KEY_RIGHTALT";
    } else if (upper == "ALT_L") {
        upper = "KEY_LEFTALT";
    } else if (upper == "CTRL_L") {
        upper = "KEY_LEFTCTRL";
    } else if (upper == "CTRL_R") {
        upper = "KEY_RIGHTCTRL";
    } else if (upper == "SHIFT_L") {
        upper = "KEY_LEFTSHIFT";
    } else if (upper == "SHIFT_R") {
        upper = "KEY_RIGHTSHIFT";
    } else if (upper == "HANGUL") {
        upper = "KEY_HANGUL";
    } else if (upper.rfind("KEY_", 0) != 0) {
        upper = "KEY_" + upper;
    }

    const auto& table = keycode_table();
    auto it = table.find(upper);
    if (it == table.end()) {
        throw ConfigError("Unknown key code '" + name + "' in " + source);
    }
    return it->second;
}

}  // namespace

ConfigError::ConfigError(const std::string& what) : std::runtime_error(what) {}

ToggleConfig default_toggle_config() {
    ToggleConfig config;
    config.toggle_keys = {KEY_RIGHTALT, KEY_HANGEUL};
    config.default_mode = InputMode::Hangul;
    return config;
}

ToggleConfig load_toggle_config(const std::string& path) {
    std::ifstream stream(path);
    if (!stream.is_open()) {
        throw ConfigError("Failed to open toggle config: " + path);
    }

    std::string line;
    bool in_toggle = false;
    std::string keys_value;
    std::string mode_value;

    while (std::getline(stream, line)) {
        std::string trimmed = trim_copy(line);
        if (trimmed.empty()) {
            continue;
        }
        if (trimmed[0] == '#' || trimmed[0] == ';') {
            continue;
        }
        if (trimmed.front() == '[' && trimmed.back() == ']') {
            std::string section = trim_copy(trimmed.substr(1, trimmed.size() - 2));
            in_toggle = (section == "toggle");
            continue;
        }
        if (!in_toggle) {
            continue;
        }
        auto eq_pos = trimmed.find('=');
        if (eq_pos == std::string::npos) {
            throw ConfigError("Invalid line in " + path + ": " + trimmed);
        }
        std::string key = trim_copy(trimmed.substr(0, eq_pos));
        std::string value = trim_copy(trimmed.substr(eq_pos + 1));
        if (key == "keys") {
            keys_value = value;
        } else if (key == "default_mode") {
            mode_value = value;
        }
    }

    if (keys_value.empty()) {
        throw ConfigError("No toggle keys defined in " + path);
    }

    auto tokens = split_comma(keys_value);
    if (tokens.empty()) {
        throw ConfigError("No toggle keys defined in " + path);
    }

    ToggleConfig config;
    for (const auto& token : tokens) {
        config.toggle_keys.push_back(parse_keycode(token, path));
    }

    if (!mode_value.empty()) {
        std::string lower = mode_value;
        std::transform(lower.begin(), lower.end(), lower.begin(), [](unsigned char ch) {
            return static_cast<char>(std::tolower(ch));
        });
        if (lower == "hangul") {
            config.default_mode = InputMode::Hangul;
        } else if (lower == "latin") {
            config.default_mode = InputMode::Latin;
        } else {
            throw ConfigError("Invalid default_mode '" + mode_value + "' in " + path);
        }
    } else {
        config.default_mode = InputMode::Hangul;
    }

    return config;
}

}  // namespace hanfe
