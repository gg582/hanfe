#include <fcntl.h>
#include <unistd.h>

#include <cstring>
#include <filesystem>
#include <iostream>
#include <memory>
#include <stdexcept>

#include "hanfe/cli.hpp"
#include "hanfe/config.hpp"
#include "hanfe/engine.hpp"
#include "hanfe/layout.hpp"

namespace hanfe {
namespace {

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

        if (options.device_path.empty()) {
            std::cerr << "Error: --device is required.\n";
            print_usage();
            return 1;
        }

        Layout layout = load_layout(options.layout_name);
        ToggleConfig toggle = resolve_toggle_config(options);

        int fd_raw = ::open(options.device_path.c_str(), O_RDONLY | O_NONBLOCK);
        if (fd_raw < 0) {
            throw std::runtime_error("Failed to open device '" + options.device_path + "': " +
                                     std::string(strerror(errno)));
        }
        FdHolder device_fd(fd_raw);

        auto emitter = std::make_unique<FallbackEmitter>(unicode_hex_keycodes(), options.tty_path);
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
