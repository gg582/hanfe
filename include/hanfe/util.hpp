#pragma once

#include <cstdint>
#include <string>
#include <string_view>
#include <vector>

namespace hanfe {

std::string utf8_from_char32(char32_t codepoint);
std::string utf8_from_u32string(const std::u32string& value);
std::u32string utf8_to_u32(const std::string& value);

std::string trim_copy(std::string_view text);
std::vector<std::string> split_comma(std::string_view text);

}  // namespace hanfe
