#include "hanfe/device.hpp"

#include <fcntl.h>
#include <linux/input-event-codes.h>
#include <linux/input.h>
#include <sys/ioctl.h>
#include <unistd.h>

#include <algorithm>
#include <array>
#include <cctype>
#include <cerrno>
#include <filesystem>
#include <optional>
#include <string>
#include <system_error>
#include <unordered_set>
#include <vector>

namespace hanfe {
namespace {

constexpr size_t kBitsPerLong = sizeof(unsigned long) * 8;

constexpr size_t bits_to_longs(size_t bits) {
    return (bits + kBitsPerLong - 1) / kBitsPerLong;
}

constexpr size_t kEventBitsSize = bits_to_longs(EV_MAX + 1);
constexpr size_t kKeyBitsSize = bits_to_longs(KEY_MAX + 1);

template <size_t N>
bool test_bit(const std::array<unsigned long, N>& bits, int bit) {
    size_t index = static_cast<size_t>(bit) / kBitsPerLong;
    size_t offset = static_cast<size_t>(bit) % kBitsPerLong;
    if (index >= bits.size()) {
        return false;
    }
    return (bits[index] & (1UL << offset)) != 0;
}

std::string to_lower_copy(const std::string& text) {
    std::string lower;
    lower.reserve(text.size());
    for (unsigned char ch : text) {
        lower.push_back(static_cast<char>(std::tolower(ch)));
    }
    return lower;
}

bool looks_like_keyboard_name(const std::string& filename) {
    std::string lower = to_lower_copy(filename);
    return lower.find("kbd") != std::string::npos || lower.find("keyboard") != std::string::npos;
}

std::vector<std::string> collect_keyboard_symlinks(const std::filesystem::path& dir) {
    std::vector<std::string> entries;
    std::error_code ec;
    if (!std::filesystem::exists(dir, ec) || ec) {
        return entries;
    }
    std::filesystem::directory_iterator it(dir, ec);
    if (ec) {
        return entries;
    }
    std::filesystem::directory_iterator end;
    for (; it != end; it.increment(ec)) {
        if (ec) {
            break;
        }
        const auto& entry = *it;
        auto name = entry.path().filename().string();
        if (!looks_like_keyboard_name(name)) {
            continue;
        }
        entries.push_back(entry.path().string());
    }
    std::sort(entries.begin(), entries.end());
    entries.erase(std::unique(entries.begin(), entries.end()), entries.end());
    return entries;
}

std::vector<std::string> collect_event_nodes() {
    std::vector<std::string> entries;
    std::error_code ec;
    const std::filesystem::path dir{"/dev/input"};
    if (!std::filesystem::exists(dir, ec) || ec) {
        return entries;
    }
    std::filesystem::directory_iterator it(dir, ec);
    if (ec) {
        return entries;
    }
    std::filesystem::directory_iterator end;
    for (; it != end; it.increment(ec)) {
        if (ec) {
            break;
        }
        const auto& entry = *it;
        auto name = entry.path().filename().string();
        if (name.rfind("event", 0) == 0) {
            entries.push_back(entry.path().string());
        }
    }
    std::sort(entries.begin(), entries.end());
    entries.erase(std::unique(entries.begin(), entries.end()), entries.end());
    return entries;
}

std::vector<std::string> gather_candidate_paths() {
    std::vector<std::string> candidates;
    std::unordered_set<std::string> seen;

    auto append_unique = [&](const std::vector<std::string>& items) {
        for (const auto& item : items) {
            if (seen.insert(item).second) {
                candidates.push_back(item);
            }
        }
    };

    append_unique(collect_keyboard_symlinks("/dev/input/by-id"));
    append_unique(collect_keyboard_symlinks("/dev/input/by-path"));
    append_unique(collect_event_nodes());

    return candidates;
}

bool is_keyboard_fd(int fd) {
    std::array<unsigned long, kEventBitsSize> ev_bits{};
    if (ioctl(fd, EVIOCGBIT(0, ev_bits.size() * sizeof(unsigned long)), ev_bits.data()) < 0) {
        return false;
    }
    if (!test_bit(ev_bits, EV_KEY)) {
        return false;
    }

    std::array<unsigned long, kKeyBitsSize> key_bits{};
    if (ioctl(fd, EVIOCGBIT(EV_KEY, key_bits.size() * sizeof(unsigned long)), key_bits.data()) < 0) {
        return false;
    }

    const int required_keys[] = {KEY_A, KEY_Z, KEY_SPACE, KEY_ENTER, KEY_LEFTSHIFT};
    for (int code : required_keys) {
        if (!test_bit(key_bits, code)) {
            return false;
        }
    }
    return true;
}

std::string read_device_name(int fd) {
    char buffer[256] = {0};
    if (ioctl(fd, EVIOCGNAME(sizeof(buffer)), buffer) < 0) {
        return {};
    }
    return std::string(buffer);
}

struct FdCloser {
    int fd;
    explicit FdCloser(int value) : fd(value) {}
    ~FdCloser() {
        if (fd >= 0) {
            ::close(fd);
        }
    }
    FdCloser(const FdCloser&) = delete;
    FdCloser& operator=(const FdCloser&) = delete;
};

}  // namespace

std::vector<DetectedDevice> list_keyboard_devices(std::string* error_message) {
    std::vector<DetectedDevice> devices;
    auto candidates = gather_candidate_paths();
    bool permission_denied = false;
    std::string last_error;

    for (const auto& path : candidates) {
        int fd = ::open(path.c_str(), O_RDONLY | O_NONBLOCK);
        if (fd < 0) {
            if (errno == EACCES || errno == EPERM) {
                permission_denied = true;
            }
            last_error = path + ": " + std::system_category().message(errno);
            continue;
        }
        FdCloser closer{fd};
        if (!is_keyboard_fd(fd)) {
            continue;
        }
        DetectedDevice device;
        device.path = path;
        device.name = read_device_name(fd);
        devices.push_back(std::move(device));
    }

    if (devices.empty() && error_message) {
        if (permission_denied) {
            *error_message =
                "Permission denied while probing input devices. Try running as root or adjusting udev permissions.";
        } else if (candidates.empty()) {
            *error_message = "No evdev devices found under /dev/input.";
        } else if (!last_error.empty()) {
            *error_message = "No keyboard-like device found. Last error: " + last_error;
        } else {
            *error_message = "No keyboard-like device found.";
        }
    }

    return devices;
}

std::optional<DetectedDevice> detect_keyboard_device(std::string* error_message) {
    auto devices = list_keyboard_devices(error_message);
    if (!devices.empty()) {
        return devices.front();
    }
    return std::nullopt;
}

}  // namespace hanfe
