#pragma once

#include <optional>
#include <string>
#include <vector>

namespace hanfe {

struct DetectedDevice {
    std::string path;
    std::string name;
};

std::vector<DetectedDevice> list_keyboard_devices(std::string* error_message = nullptr);
std::optional<DetectedDevice> detect_keyboard_device(std::string* error_message = nullptr);

}  // namespace hanfe
