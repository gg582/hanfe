#pragma once

#include <optional>
#include <string>
#include <unordered_map>
#include <vector>

#include "hanfe/composer.hpp"

namespace hanfe {

enum class SymbolKind {
    Jamo,
    Text,
    Passthrough,
};

struct LayoutSymbol {
    SymbolKind kind = SymbolKind::Passthrough;
    std::string text;
    char32_t jamo = U'\0';
    JamoRole role = JamoRole::Auto;
    bool commit_before = false;
};

struct LayoutEntry {
    std::optional<LayoutSymbol> normal;
    std::optional<LayoutSymbol> shifted;
};

class Layout {
   public:
    Layout() = default;
    Layout(std::string name, std::unordered_map<int, LayoutEntry> mapping);

    const LayoutSymbol* translate(int keycode, bool shift_active) const;
    const std::string& name() const { return name_; }

   private:
    std::string name_;
    std::unordered_map<int, LayoutEntry> mapping_;
};

Layout make_dubeolsik_layout();
Layout make_sebeolsik_390_layout();

std::vector<std::string> available_layouts();
Layout load_layout(const std::string& name);

std::unordered_map<char, int> unicode_hex_keycodes();

}  // namespace hanfe
