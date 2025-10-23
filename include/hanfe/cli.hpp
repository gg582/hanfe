#pragma once

#include <optional>
#include <string>
#include <vector>

namespace hanfe {

struct CliOptions {
    bool show_help = false;
    bool list_layouts = false;
    std::string device_path;
    std::string layout_name = "dubeolsik";
    std::optional<std::string> toggle_config_path;
    std::optional<std::string> tty_path;
};

CliOptions parse_arguments(int argc, char** argv);
void print_usage();

}  // namespace hanfe
