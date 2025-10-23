#include "hanfe/cli.hpp"

#include <iostream>
#include <stdexcept>

namespace hanfe {
namespace {

std::string extract_value(const std::string& arg, int& index, int argc, char** argv,
                          const std::string& name) {
    auto eq = arg.find('=');
    if (eq != std::string::npos) {
        return arg.substr(eq + 1);
    }
    if (index + 1 >= argc) {
        throw std::runtime_error("Option " + name + " requires a value");
    }
    ++index;
    return std::string(argv[index]);
}

}  // namespace

CliOptions parse_arguments(int argc, char** argv) {
    CliOptions options;
    for (int i = 1; i < argc; ++i) {
        std::string arg = argv[i];
        if (arg == "--help" || arg == "-h") {
            options.show_help = true;
        } else if (arg == "--list-layouts") {
            options.list_layouts = true;
        } else if (arg.rfind("--device", 0) == 0) {
            options.device_path = extract_value(arg, i, argc, argv, "--device");
        } else if (arg.rfind("--layout", 0) == 0) {
            options.layout_name = extract_value(arg, i, argc, argv, "--layout");
        } else if (arg.rfind("--toggle-config", 0) == 0) {
            options.toggle_config_path = extract_value(arg, i, argc, argv, "--toggle-config");
        } else if (arg.rfind("--tty", 0) == 0) {
            options.tty_path = extract_value(arg, i, argc, argv, "--tty");
        } else {
            throw std::runtime_error("Unknown option: " + arg);
        }
    }
    return options;
}

void print_usage() {
    std::cout << "hanfe - Hangul IME interceptor\n";
    std::cout << "Usage: hanfe --device /dev/input/eventX [options]\n\n";
    std::cout << "Options:\n";
    std::cout << "  --device PATH           Path to the evdev keyboard device (required)\n";
    std::cout << "  --layout NAME           Keyboard layout (default: dubeolsik)\n";
    std::cout << "  --toggle-config PATH    Path to toggle.ini (default: ./toggle.ini if present)\n";
    std::cout << "  --tty PATH              Optional TTY to mirror text output to\n";
    std::cout << "  --list-layouts          List available layouts\n";
    std::cout << "  -h, --help              Show this help message\n";
}

}  // namespace hanfe
