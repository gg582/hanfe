#include <fcntl.h>
#include <unistd.h>

#include <cerrno>
#include <filesystem>
#include <iostream>
#include <memory>
#include <stdexcept>
#include <system_error>
#include <cstring>

#include "hanfe/cli.hpp"
#include "hanfe/config.hpp"
#include "hanfe/device.hpp"
#include "hanfe/engine.hpp"
#include "hanfe/emitter.h"
#include "hanfe/layout.hpp"

namespace hanfe {
namespace {

int hex_index(char ch) {
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

ToggleConfig resolve_toggle_config(const CliOptions& options) {
    if (options.toggle_config_path) {
        return load_toggle_config(*options.toggle_config_path);
    }
    std::filesystem::path default_path = std::filesystem::current_path() / "toggle.ini";
    if (std::filesystem::exists(default_path)) {
        return load_toggle_config(default_path.string());
    }
    return default_toggle_config();
}

struct FdHolder {
    int fd = -1;
    explicit FdHolder(int value) : fd(value) {}
    ~FdHolder() {
        if (fd >= 0) {
            ::close(fd);
        }
    }
    FdHolder(const FdHolder&) = delete;
    FdHolder& operator=(const FdHolder&) = delete;
    FdHolder(FdHolder&& other) noexcept : fd(other.fd) { other.fd = -1; }
    FdHolder& operator=(FdHolder&& other) noexcept {
        if (this != &other) {
            if (fd >= 0) {
                ::close(fd);
            }
            fd = other.fd;
            other.fd = -1;
        }
        return *this;
    }
};

}  // namespace
}  // namespace hanfe

int main(int argc, char** argv) {
    using namespace hanfe;
    try {
        CliOptions options = parse_arguments(argc, argv);
        if (options.show_help) {
            print_usage();
            std::cout << "\nAvailable layouts:\n";
            for (const auto& name : available_layouts()) {
                std::cout << "  " << name << '\n';
            }
            return 0;
        }

        if (options.list_layouts) {
            for (const auto& name : available_layouts()) {
                std::cout << name << '\n';
            }
            return 0;
        }

        std::string device_path = options.device_path;
        if (device_path.empty()) {
            std::string detection_error;
            auto detected = detect_keyboard_device(&detection_error);
            if (!detected) {
                std::cerr << "Error: failed to auto-detect a keyboard device";
                if (!detection_error.empty()) {
                    std::cerr << ": " << detection_error;
                }
                std::cerr << "\nProvide --device /dev/input/eventX explicitly.\n";
                return 1;
            }
            device_path = detected->path;
            std::cout << "Auto-detected keyboard device: " << device_path;
            if (!detected->name.empty()) {
                std::cout << " [" << detected->name << "]";
            }
            std::cout << '\n';
        }

        Layout layout = load_layout(options.layout_name);
        ToggleConfig toggle = resolve_toggle_config(options);

        int fd_raw = ::open(device_path.c_str(), O_RDONLY | O_NONBLOCK);
        if (fd_raw < 0) {
            std::error_code ec(errno, std::system_category());
            throw std::runtime_error("Failed to open device '" + device_path + "': " + ec.message());
        }
        FdHolder device_fd(fd_raw);

        int hex_keycodes[16];
        for (int i = 0; i < 16; ++i) {
            hex_keycodes[i] = -1;
        }
        auto hex_map = unicode_hex_keycodes();
        for (const auto& entry : hex_map) {
            int idx = hex_index(entry.first);
            if (idx >= 0) {
                hex_keycodes[idx] = entry.second;
            }
        }

        const char* tty_path = options.tty_path ? options.tty_path->c_str() : nullptr;
        fallback_emitter* raw_emitter = fallback_emitter_open(hex_keycodes, tty_path);
        if (!raw_emitter) {
            int err = errno;
            throw std::runtime_error("Failed to create fallback emitter: " +
                                     std::string(std::strerror(err)));
        }
        FallbackEmitterPtr emitter(raw_emitter);
        HanfeEngine engine(device_fd.fd, std::move(layout), std::move(toggle), std::move(emitter));
        engine.run();
    } catch (const ConfigError& err) {
        std::cerr << "Configuration error: " << err.what() << '\n';
        return 2;
    } catch (const std::exception& err) {
        std::cerr << "Error: " << err.what() << '\n';
        return 1;
    }
    return 0;
}
