#include "hanfe/layout.hpp"

#include <linux/input-event-codes.h>

#include <algorithm>
#include <stdexcept>

namespace hanfe {
namespace {

LayoutSymbol make_text_symbol(const std::string& value, bool commit_before = true) {
    LayoutSymbol symbol;
    symbol.kind = SymbolKind::Text;
    symbol.text = value;
    symbol.commit_before = commit_before;
    return symbol;
}

LayoutSymbol make_jamo_symbol(char32_t value, JamoRole role = JamoRole::Auto) {
    LayoutSymbol symbol;
    symbol.kind = SymbolKind::Jamo;
    symbol.jamo = value;
    symbol.role = role;
    symbol.commit_before = false;
    return symbol;
}

LayoutSymbol make_passthrough_symbol(bool commit_before = true) {
    LayoutSymbol symbol;
    symbol.kind = SymbolKind::Passthrough;
    symbol.commit_before = commit_before;
    return symbol;
}

Layout build_dubeolsik() {
    std::unordered_map<int, LayoutEntry> mapping;

    auto add = [&](int key, LayoutSymbol normal, std::optional<LayoutSymbol> shifted = std::nullopt) {
        LayoutEntry entry;
        entry.normal = normal;
        entry.shifted = shifted;
        mapping.emplace(key, entry);
    };

    add(KEY_Q, make_jamo_symbol(U'ㅂ'), make_jamo_symbol(U'ㅃ'));
    add(KEY_W, make_jamo_symbol(U'ㅈ'), make_jamo_symbol(U'ㅉ'));
    add(KEY_E, make_jamo_symbol(U'ㄷ'), make_jamo_symbol(U'ㄸ'));
    add(KEY_R, make_jamo_symbol(U'ㄱ'), make_jamo_symbol(U'ㄲ'));
    add(KEY_T, make_jamo_symbol(U'ㅅ'), make_jamo_symbol(U'ㅆ'));
    add(KEY_Y, make_jamo_symbol(U'ㅛ'));
    add(KEY_U, make_jamo_symbol(U'ㅕ'));
    add(KEY_I, make_jamo_symbol(U'ㅑ'));
    add(KEY_O, make_jamo_symbol(U'ㅐ'), make_jamo_symbol(U'ㅒ'));
    add(KEY_P, make_jamo_symbol(U'ㅔ'), make_jamo_symbol(U'ㅖ'));
    add(KEY_A, make_jamo_symbol(U'ㅁ'));
    add(KEY_S, make_jamo_symbol(U'ㄴ'));
    add(KEY_D, make_jamo_symbol(U'ㅇ'));
    add(KEY_F, make_jamo_symbol(U'ㄹ'));
    add(KEY_G, make_jamo_symbol(U'ㅎ'));
    add(KEY_H, make_jamo_symbol(U'ㅗ'));
    add(KEY_J, make_jamo_symbol(U'ㅓ'));
    add(KEY_K, make_jamo_symbol(U'ㅏ'));
    add(KEY_L, make_jamo_symbol(U'ㅣ'));

    {
        LayoutEntry entry;
        entry.normal = make_text_symbol(";");
        entry.shifted = make_text_symbol(":");
        mapping.emplace(KEY_SEMICOLON, entry);
    }
    {
        LayoutEntry entry;
        entry.normal = make_text_symbol("'");
        entry.shifted = make_text_symbol("\"");
        mapping.emplace(KEY_APOSTROPHE, entry);
    }

    add(KEY_Z, make_jamo_symbol(U'ㅋ'));
    add(KEY_X, make_jamo_symbol(U'ㅌ'));
    add(KEY_C, make_jamo_symbol(U'ㅊ'));
    add(KEY_V, make_jamo_symbol(U'ㅍ'));
    add(KEY_B, make_jamo_symbol(U'ㅠ'));
    add(KEY_N, make_jamo_symbol(U'ㅜ'));
    add(KEY_M, make_jamo_symbol(U'ㅡ'));

    {
        LayoutEntry entry;
        entry.normal = make_text_symbol(",");
        entry.shifted = make_text_symbol("<");
        mapping.emplace(KEY_COMMA, entry);
    }
    {
        LayoutEntry entry;
        entry.normal = make_text_symbol(".");
        entry.shifted = make_text_symbol(">");
        mapping.emplace(KEY_DOT, entry);
    }
    {
        LayoutEntry entry;
        entry.normal = make_text_symbol("/");
        entry.shifted = make_text_symbol("?");
        mapping.emplace(KEY_SLASH, entry);
    }

    add(KEY_SPACE, make_text_symbol(" "));

    auto add_number = [&](int key, const std::string& normal, const std::string& shifted) {
        LayoutEntry entry;
        entry.normal = make_text_symbol(normal);
        entry.shifted = make_text_symbol(shifted);
        mapping.emplace(key, entry);
    };

    add_number(KEY_1, "1", "!");
    add_number(KEY_2, "2", "@");
    add_number(KEY_3, "3", "#");
    add_number(KEY_4, "4", "$");
    add_number(KEY_5, "5", "%");
    add_number(KEY_6, "6", "^");
    add_number(KEY_7, "7", "&");
    add_number(KEY_8, "8", "*");
    add_number(KEY_9, "9", "(");
    add_number(KEY_0, "0", ")");
    add_number(KEY_MINUS, "-", "_");
    add_number(KEY_EQUAL, "=", "+");
    add_number(KEY_LEFTBRACE, "[", "{");
    add_number(KEY_RIGHTBRACE, "]", "}");

    {
        LayoutEntry entry;
        entry.normal = make_text_symbol("\\");
        entry.shifted = make_text_symbol("|");
        mapping.emplace(KEY_BACKSLASH, entry);
    }
    {
        LayoutEntry entry;
        entry.normal = make_text_symbol("`");
        entry.shifted = make_text_symbol("~");
        mapping.emplace(KEY_GRAVE, entry);
    }

    mapping.emplace(KEY_TAB, LayoutEntry{make_passthrough_symbol(), std::nullopt});
    mapping.emplace(KEY_ENTER, LayoutEntry{make_passthrough_symbol(), std::nullopt});
    mapping.emplace(KEY_ESC, LayoutEntry{make_passthrough_symbol(), std::nullopt});
    mapping.emplace(KEY_BACKSPACE, LayoutEntry{make_passthrough_symbol(), std::nullopt});

    return Layout("dubeolsik", std::move(mapping));
}

Layout build_sebeolsik_390() {
    std::unordered_map<int, LayoutEntry> mapping;

    auto set_entry = [&](int key, const LayoutSymbol& normal, std::optional<LayoutSymbol> shifted) {
        LayoutEntry entry;
        entry.normal = normal;
        entry.shifted = shifted;
        mapping.emplace(key, entry);
    };

    auto text = [](const std::string& value) { return make_text_symbol(value); };
    auto jamo = [](char32_t value) { return make_jamo_symbol(value); };

    set_entry(KEY_GRAVE, text("`"), make_text_symbol("~"));
    set_entry(KEY_1, text("1"), make_text_symbol("!"));
    set_entry(KEY_2, text("2"), make_text_symbol("@"));
    set_entry(KEY_3, text("3"), make_text_symbol("#"));
    set_entry(KEY_4, text("4"), make_text_symbol("$"));
    set_entry(KEY_5, text("5"), make_text_symbol("%"));
    set_entry(KEY_6, text("6"), make_text_symbol("^"));
    set_entry(KEY_7, text("7"), make_text_symbol("&"));
    set_entry(KEY_8, text("8"), make_text_symbol("*"));
    set_entry(KEY_9, text("9"), make_text_symbol("("));
    set_entry(KEY_0, text("0"), make_text_symbol(")"));
    set_entry(KEY_MINUS, text("-"), make_text_symbol("_"));
    set_entry(KEY_EQUAL, text("="), make_text_symbol("+"));

    set_entry(KEY_Q, jamo(U'ㅂ'), make_jamo_symbol(U'ㅃ'));
    set_entry(KEY_W, jamo(U'ㅈ'), make_jamo_symbol(U'ㅉ'));
    set_entry(KEY_E, jamo(U'ㄷ'), make_jamo_symbol(U'ㄸ'));
    set_entry(KEY_R, jamo(U'ㄱ'), make_jamo_symbol(U'ㄲ'));
    set_entry(KEY_T, jamo(U'ㅅ'), make_jamo_symbol(U'ㅆ'));
    set_entry(KEY_Y, jamo(U'ㅛ'), make_jamo_symbol(U'ㅅ', JamoRole::Trailing));
    set_entry(KEY_U, jamo(U'ㅕ'), make_jamo_symbol(U'ㅈ', JamoRole::Trailing));
    set_entry(KEY_I, jamo(U'ㅑ'), make_jamo_symbol(U'ㅊ', JamoRole::Trailing));
    set_entry(KEY_O, jamo(U'ㅐ'), make_jamo_symbol(U'ㅋ', JamoRole::Trailing));
    set_entry(KEY_P, jamo(U'ㅔ'), make_jamo_symbol(U'ㅌ', JamoRole::Trailing));
    set_entry(KEY_LEFTBRACE, jamo(U'ㅒ'), make_jamo_symbol(U'ㅍ', JamoRole::Trailing));
    set_entry(KEY_RIGHTBRACE, jamo(U'ㅖ'), make_jamo_symbol(U'ㅎ', JamoRole::Trailing));
    {
        LayoutEntry entry;
        entry.normal = jamo(U'ㅢ');
        entry.shifted = make_text_symbol("|");
        mapping.emplace(KEY_BACKSLASH, entry);
    }

    set_entry(KEY_A, jamo(U'ㅁ'), make_jamo_symbol(U'ㅁ'));
    set_entry(KEY_S, jamo(U'ㄴ'), make_jamo_symbol(U'ㄴ'));
    set_entry(KEY_D, jamo(U'ㅇ'), make_jamo_symbol(U'ㅇ'));
    set_entry(KEY_F, jamo(U'ㄹ'), make_jamo_symbol(U'ㄹ'));
    set_entry(KEY_G, jamo(U'ㅎ'), make_jamo_symbol(U'ㅎ'));
    set_entry(KEY_H, jamo(U'ㅗ'), make_jamo_symbol(U'ㄱ', JamoRole::Trailing));
    set_entry(KEY_J, jamo(U'ㅓ'), make_jamo_symbol(U'ㄴ', JamoRole::Trailing));
    set_entry(KEY_K, jamo(U'ㅏ'), make_jamo_symbol(U'ㄷ', JamoRole::Trailing));
    set_entry(KEY_L, jamo(U'ㅣ'), make_jamo_symbol(U'ㄹ', JamoRole::Trailing));
    set_entry(KEY_SEMICOLON, jamo(U'ㅠ'), make_jamo_symbol(U'ㅁ', JamoRole::Trailing));
    set_entry(KEY_APOSTROPHE, jamo(U'ㅜ'), make_jamo_symbol(U'ㅂ', JamoRole::Trailing));

    set_entry(KEY_Z, jamo(U'ㅋ'), make_jamo_symbol(U'ㅋ'));
    set_entry(KEY_X, jamo(U'ㅌ'), make_jamo_symbol(U'ㅌ'));
    set_entry(KEY_C, jamo(U'ㅊ'), make_jamo_symbol(U'ㅊ'));
    set_entry(KEY_V, jamo(U'ㅍ'), make_jamo_symbol(U'ㅍ'));
    set_entry(KEY_B, jamo(U'ㅠ'), make_jamo_symbol(U'ㅇ', JamoRole::Trailing));
    set_entry(KEY_N, jamo(U'ㅜ'), make_jamo_symbol(U'ㅅ', JamoRole::Trailing));
    set_entry(KEY_M, jamo(U'ㅡ'), make_jamo_symbol(U'ㅎ', JamoRole::Trailing));

    {
        LayoutEntry entry;
        entry.normal = jamo(U'ㅘ');
        entry.shifted = jamo(U'ㅙ');
        mapping.emplace(KEY_COMMA, entry);
    }
    {
        LayoutEntry entry;
        entry.normal = jamo(U'ㅝ');
        entry.shifted = jamo(U'ㅞ');
        mapping.emplace(KEY_DOT, entry);
    }
    {
        LayoutEntry entry;
        entry.normal = jamo(U'ㅟ');
        entry.shifted = make_text_symbol("?");
        mapping.emplace(KEY_SLASH, entry);
    }

    set_entry(KEY_SPACE, text(" "), std::nullopt);

    mapping.emplace(KEY_ENTER, LayoutEntry{make_passthrough_symbol(), std::nullopt});
    mapping.emplace(KEY_TAB, LayoutEntry{make_passthrough_symbol(), std::nullopt});
    mapping.emplace(KEY_ESC, LayoutEntry{make_passthrough_symbol(), std::nullopt});
    mapping.emplace(KEY_BACKSPACE, LayoutEntry{make_passthrough_symbol(), std::nullopt});

    return Layout("sebeolsik-390", std::move(mapping));
}

}  // namespace

Layout::Layout(std::string name, std::unordered_map<int, LayoutEntry> mapping)
    : name_(std::move(name)), mapping_(std::move(mapping)) {}

const LayoutSymbol* Layout::translate(int keycode, bool shift_active) const {
    auto it = mapping_.find(keycode);
    if (it == mapping_.end()) {
        return nullptr;
    }
    const LayoutEntry& entry = it->second;
    if (shift_active && entry.shifted) {
        return &entry.shifted.value();
    }
    if (entry.normal) {
        return &entry.normal.value();
    }
    if (entry.shifted) {
        return &entry.shifted.value();
    }
    return nullptr;
}

Layout make_dubeolsik_layout() { return build_dubeolsik(); }

Layout make_sebeolsik_390_layout() { return build_sebeolsik_390(); }

std::vector<std::string> available_layouts() {
    std::vector<std::string> names = {"dubeolsik", "sebeolsik-390"};
    std::sort(names.begin(), names.end());
    return names;
}

Layout load_layout(const std::string& name) {
    if (name == "dubeolsik") {
        return make_dubeolsik_layout();
    }
    if (name == "sebeolsik-390") {
        return make_sebeolsik_390_layout();
    }
    throw std::runtime_error("Unknown layout: " + name);
}

std::unordered_map<char, int> unicode_hex_keycodes() {
    return {
        {'0', KEY_0}, {'1', KEY_1}, {'2', KEY_2}, {'3', KEY_3}, {'4', KEY_4}, {'5', KEY_5},
        {'6', KEY_6}, {'7', KEY_7}, {'8', KEY_8}, {'9', KEY_9}, {'a', KEY_A}, {'b', KEY_B},
        {'c', KEY_C}, {'d', KEY_D}, {'e', KEY_E}, {'f', KEY_F}};
}

}  // namespace hanfe
