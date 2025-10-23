#pragma once

#include <stdexcept>
#include <string>
#include <vector>

namespace hanfe {

enum class InputMode {
    Hangul,
    Latin,
};

struct ToggleConfig {
    std::vector<int> toggle_keys;
    InputMode default_mode = InputMode::Hangul;
};

class ConfigError : public std::runtime_error {
   public:
    explicit ConfigError(const std::string& what);
};

ToggleConfig load_toggle_config(const std::string& path);
ToggleConfig default_toggle_config();

}  // namespace hanfe
