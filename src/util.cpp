#include "hanfe/util.hpp"

#include <cctype>

namespace hanfe {
namespace {
bool is_space(char ch) {
    return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == '\f' || ch == '\v';
}
}

std::string utf8_from_char32(char32_t codepoint) {
    std::string out;
    if (codepoint <= 0x7F) {
        out.push_back(static_cast<char>(codepoint));
    } else if (codepoint <= 0x7FF) {
        out.push_back(static_cast<char>(0xC0 | ((codepoint >> 6) & 0x1F)));
        out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    } else if (codepoint <= 0xFFFF) {
        out.push_back(static_cast<char>(0xE0 | ((codepoint >> 12) & 0x0F)));
        out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    } else {
        out.push_back(static_cast<char>(0xF0 | ((codepoint >> 18) & 0x07)));
        out.push_back(static_cast<char>(0x80 | ((codepoint >> 12) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
        out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    }
    return out;
}

std::string utf8_from_u32string(const std::u32string& value) {
    std::string out;
    for (char32_t ch : value) {
        out += utf8_from_char32(ch);
    }
    return out;
}

std::u32string utf8_to_u32(const std::string& value) {
    std::u32string out;
    size_t i = 0;
    while (i < value.size()) {
        unsigned char lead = static_cast<unsigned char>(value[i]);
        if (lead < 0x80) {
            out.push_back(lead);
            ++i;
            continue;
        }
        size_t extra = 0;
        char32_t codepoint = 0;
        if ((lead & 0xE0) == 0xC0) {
            extra = 1;
            codepoint = lead & 0x1F;
        } else if ((lead & 0xF0) == 0xE0) {
            extra = 2;
            codepoint = lead & 0x0F;
        } else if ((lead & 0xF8) == 0xF0) {
            extra = 3;
            codepoint = lead & 0x07;
        } else {
            // Skip invalid byte
            ++i;
            continue;
        }
        if (i + extra >= value.size()) {
            break;
        }
        bool valid = true;
        for (size_t j = 0; j < extra; ++j) {
            unsigned char trail = static_cast<unsigned char>(value[i + 1 + j]);
            if ((trail & 0xC0) != 0x80) {
                valid = false;
                break;
            }
            codepoint = (codepoint << 6) | (trail & 0x3F);
        }
        if (!valid) {
            ++i;
            continue;
        }
        out.push_back(codepoint);
        i += extra + 1;
    }
    return out;
}

std::string trim_copy(std::string_view text) {
    size_t start = 0;
    while (start < text.size() && is_space(text[start])) {
        ++start;
    }
    size_t end = text.size();
    while (end > start && is_space(text[end - 1])) {
        --end;
    }
    return std::string{text.substr(start, end - start)};
}

std::vector<std::string> split_comma(std::string_view text) {
    std::vector<std::string> parts;
    std::string current;
    for (char ch : text) {
        if (ch == ',') {
            std::string trimmed = trim_copy(current);
            if (!trimmed.empty()) {
                parts.push_back(trimmed);
            }
            current.clear();
        } else {
            current.push_back(ch);
        }
    }
    std::string trimmed = trim_copy(current);
    if (!trimmed.empty()) {
        parts.push_back(trimmed);
    }
    return parts;
}

}  // namespace hanfe
