#include "hanfe/composer.hpp"

#include <unordered_map>
#include <unordered_set>

namespace hanfe {
namespace {

struct PairHash {
    size_t operator()(const std::pair<char32_t, char32_t>& pair) const noexcept {
        return static_cast<size_t>(pair.first) * 1315423911u + static_cast<size_t>(pair.second);
    }
};

const char32_t CHO_LIST[] = {U'ㄱ', U'ㄲ', U'ㄴ', U'ㄷ', U'ㄸ', U'ㄹ', U'ㅁ', U'ㅂ', U'ㅃ', U'ㅅ',
                             U'ㅆ', U'ㅇ', U'ㅈ', U'ㅉ', U'ㅊ', U'ㅋ', U'ㅌ', U'ㅍ', U'ㅎ'};

const char32_t JUNG_LIST[] = {U'ㅏ', U'ㅐ', U'ㅑ', U'ㅒ', U'ㅓ', U'ㅔ', U'ㅕ', U'ㅖ', U'ㅗ', U'ㅘ',
                              U'ㅙ', U'ㅚ', U'ㅛ', U'ㅜ', U'ㅝ', U'ㅞ', U'ㅟ', U'ㅠ', U'ㅡ', U'ㅢ',
                              U'ㅣ'};

const char32_t JONG_LIST[] = {U'\0', U'ㄱ', U'ㄲ', U'ㄳ', U'ㄴ', U'ㄵ', U'ㄶ', U'ㄷ', U'ㄹ', U'ㄺ',
                              U'ㄻ', U'ㄼ', U'ㄽ', U'ㄾ', U'ㄿ', U'ㅀ', U'ㅁ', U'ㅂ', U'ㅄ', U'ㅅ',
                              U'ㅆ', U'ㅇ', U'ㅈ', U'ㅊ', U'ㅋ', U'ㅌ', U'ㅍ', U'ㅎ'};

const std::unordered_map<std::pair<char32_t, char32_t>, char32_t, PairHash> DOUBLE_INITIAL = {
    {{U'ㄱ', U'ㄱ'}, U'ㄲ'}, {{U'ㄷ', U'ㄷ'}, U'ㄸ'}, {{U'ㅂ', U'ㅂ'}, U'ㅃ'},
    {{U'ㅈ', U'ㅈ'}, U'ㅉ'}, {{U'ㅅ', U'ㅅ'}, U'ㅆ'}};

const std::unordered_map<char32_t, std::pair<char32_t, char32_t>> INITIAL_DECOMPOSE = [] {
    std::unordered_map<char32_t, std::pair<char32_t, char32_t>> table;
    for (const auto& entry : DOUBLE_INITIAL) {
        table.emplace(entry.second, entry.first);
    }
    return table;
}();

const std::unordered_map<std::pair<char32_t, char32_t>, char32_t, PairHash> DOUBLE_MEDIAL = {
    {{U'ㅗ', U'ㅏ'}, U'ㅘ'}, {{U'ㅗ', U'ㅐ'}, U'ㅙ'}, {{U'ㅗ', U'ㅣ'}, U'ㅚ'},
    {{U'ㅜ', U'ㅓ'}, U'ㅝ'}, {{U'ㅜ', U'ㅔ'}, U'ㅞ'}, {{U'ㅜ', U'ㅣ'}, U'ㅟ'},
    {{U'ㅡ', U'ㅣ'}, U'ㅢ'}};

const std::unordered_map<char32_t, std::pair<char32_t, char32_t>> MEDIAL_DECOMPOSE = [] {
    std::unordered_map<char32_t, std::pair<char32_t, char32_t>> table;
    for (const auto& entry : DOUBLE_MEDIAL) {
        table.emplace(entry.second, entry.first);
    }
    return table;
}();

const std::unordered_map<std::pair<char32_t, char32_t>, char32_t, PairHash> DOUBLE_FINAL = {
    {{U'ㄱ', U'ㄱ'}, U'ㄲ'}, {{U'ㄱ', U'ㅅ'}, U'ㄳ'}, {{U'ㄴ', U'ㅈ'}, U'ㄵ'},
    {{U'ㄴ', U'ㅎ'}, U'ㄶ'}, {{U'ㄹ', U'ㄱ'}, U'ㄺ'}, {{U'ㄹ', U'ㅁ'}, U'ㄻ'},
    {{U'ㄹ', U'ㅂ'}, U'ㄼ'}, {{U'ㄹ', U'ㅅ'}, U'ㄽ'}, {{U'ㄹ', U'ㅌ'}, U'ㄾ'},
    {{U'ㄹ', U'ㅍ'}, U'ㄿ'}, {{U'ㄹ', U'ㅎ'}, U'ㅀ'}, {{U'ㅂ', U'ㅅ'}, U'ㅄ'},
    {{U'ㅅ', U'ㅅ'}, U'ㅆ'}};

const std::unordered_map<char32_t, std::pair<char32_t, char32_t>> FINAL_DECOMPOSE = [] {
    std::unordered_map<char32_t, std::pair<char32_t, char32_t>> table;
    for (const auto& entry : DOUBLE_FINAL) {
        table.emplace(entry.second, entry.first);
    }
    return table;
}();

const std::unordered_map<char32_t, int> CHOSEONG_INDEX = [] {
    std::unordered_map<char32_t, int> table;
    for (size_t idx = 0; idx < std::size(CHO_LIST); ++idx) {
        table.emplace(CHO_LIST[idx], static_cast<int>(idx));
    }
    return table;
}();

const std::unordered_map<char32_t, int> JUNGSEONG_INDEX = [] {
    std::unordered_map<char32_t, int> table;
    for (size_t idx = 0; idx < std::size(JUNG_LIST); ++idx) {
        table.emplace(JUNG_LIST[idx], static_cast<int>(idx));
    }
    return table;
}();

const std::unordered_map<char32_t, int> JONGSEONG_INDEX = [] {
    std::unordered_map<char32_t, int> table;
    for (size_t idx = 0; idx < std::size(JONG_LIST); ++idx) {
        table.emplace(JONG_LIST[idx], static_cast<int>(idx));
    }
    return table;
}();

const std::unordered_set<char32_t> CONSONANTS = [] {
    std::unordered_set<char32_t> set;
    for (char32_t ch : CHO_LIST) {
        set.insert(ch);
    }
    for (char32_t ch : JONG_LIST) {
        if (ch != U'\0') {
            set.insert(ch);
        }
    }
    return set;
}();

const std::unordered_set<char32_t> VOWELS = [] {
    std::unordered_set<char32_t> set;
    for (char32_t ch : JUNG_LIST) {
        set.insert(ch);
    }
    return set;
}();

bool is_consonant(char32_t ch) { return CONSONANTS.count(ch) > 0; }

bool is_vowel(char32_t ch) { return VOWELS.count(ch) > 0; }

}  // namespace

HangulComposer::HangulComposer() = default;

CompositionResult HangulComposer::feed(char32_t ch, JamoRole role) {
    std::u32string commit;
    if (is_vowel(ch)) {
        commit = handle_vowel(ch, role);
    } else {
        commit = handle_consonant(ch, role);
    }
    CompositionResult result;
    result.commit = utf8_from_u32string(commit);
    result.preedit = utf8_from_u32string(current_preedit());
    return result;
}

std::optional<std::string> HangulComposer::backspace() {
    if (trailing_) {
        auto it = FINAL_DECOMPOSE.find(*trailing_);
        if (it != FINAL_DECOMPOSE.end()) {
            trailing_ = it->second.first;
        } else {
            trailing_.reset();
        }
        return utf8_from_u32string(current_preedit());
    }
    if (vowel_) {
        auto it = MEDIAL_DECOMPOSE.find(*vowel_);
        if (it != MEDIAL_DECOMPOSE.end()) {
            vowel_ = it->second.first;
        } else {
            vowel_.reset();
            if (leading_ && *leading_ == U'ㅇ') {
                leading_.reset();
            }
        }
        return utf8_from_u32string(current_preedit());
    }
    if (leading_) {
        auto it = INITIAL_DECOMPOSE.find(*leading_);
        if (it != INITIAL_DECOMPOSE.end()) {
            leading_ = it->second.first;
        } else {
            leading_.reset();
        }
        return utf8_from_u32string(current_preedit());
    }
    return std::nullopt;
}

std::string HangulComposer::flush() {
    std::u32string commit = compose();
    leading_.reset();
    vowel_.reset();
    trailing_.reset();
    return utf8_from_u32string(commit);
}

std::u32string HangulComposer::handle_consonant(char32_t ch, JamoRole role) {
    std::u32string commit;
    bool force_trailing = role == JamoRole::Trailing;
    bool force_leading = role == JamoRole::Leading;

    if (!leading_) {
        leading_ = ch;
        trailing_.reset();
        return commit;
    }

    if (force_leading) {
        commit = compose();
        leading_ = ch;
        vowel_.reset();
        trailing_.reset();
        return commit;
    }

    if (!vowel_) {
        auto it = DOUBLE_INITIAL.find({*leading_, ch});
        if (it != DOUBLE_INITIAL.end()) {
            leading_ = it->second;
        } else {
            commit.push_back(*leading_);
            leading_ = ch;
        }
        return commit;
    }

    if (force_trailing) {
        return attach_trailing(ch);
    }

    if (!trailing_) {
        if (is_consonant(ch)) {
            trailing_ = ch;
            return commit;
        }
        commit = compose();
        leading_ = ch;
        vowel_.reset();
        trailing_.reset();
        return commit;
    }

    auto it = DOUBLE_FINAL.find({*trailing_, ch});
    if (it != DOUBLE_FINAL.end()) {
        trailing_ = it->second;
    } else {
        commit = compose();
        leading_ = ch;
        vowel_.reset();
        trailing_.reset();
    }
    return commit;
}

std::u32string HangulComposer::handle_vowel(char32_t ch, JamoRole /*role*/) {
    std::u32string commit;
    if (!leading_) {
        leading_ = U'ㅇ';
    }

    if (!vowel_) {
        vowel_ = ch;
        return commit;
    }

    auto it = DOUBLE_MEDIAL.find({*vowel_, ch});
    if (it != DOUBLE_MEDIAL.end()) {
        vowel_ = it->second;
        return commit;
    }

    if (trailing_) {
        auto split = FINAL_DECOMPOSE.find(*trailing_);
        if (split != FINAL_DECOMPOSE.end()) {
            auto [first, second] = split->second;
            trailing_ = first;
            commit = compose();
            leading_ = second;
            vowel_ = ch;
            trailing_.reset();
            return commit;
        }
        commit = compose();
        leading_ = U'ㅇ';
        vowel_ = ch;
        trailing_.reset();
        return commit;
    }

    commit = compose();
    leading_ = U'ㅇ';
    vowel_ = ch;
    trailing_.reset();
    return commit;
}

std::u32string HangulComposer::attach_trailing(char32_t ch) {
    std::u32string commit;
    if (!trailing_) {
        if (is_consonant(ch)) {
            trailing_ = ch;
            return commit;
        }
        commit = compose();
        leading_ = ch;
        vowel_.reset();
        trailing_.reset();
        return commit;
    }

    auto it = DOUBLE_FINAL.find({*trailing_, ch});
    if (it != DOUBLE_FINAL.end()) {
        trailing_ = it->second;
        return commit;
    }

    commit = compose();
    leading_ = ch;
    vowel_.reset();
    trailing_.reset();
    return commit;
}

std::u32string HangulComposer::compose() const {
    if (!leading_ && !vowel_) {
        return U"";
    }
    if (leading_ && vowel_) {
        int lead_index = CHOSEONG_INDEX.at(*leading_);
        int vowel_index = JUNGSEONG_INDEX.at(*vowel_);
        int tail_index = 0;
        if (trailing_) {
            tail_index = JONGSEONG_INDEX.at(*trailing_);
        }
        char32_t codepoint = static_cast<char32_t>(0xAC00 + ((lead_index * 21) + vowel_index) * 28 + tail_index);
        return std::u32string(1, codepoint);
    }
    if (leading_) {
        return std::u32string(1, *leading_);
    }
    if (vowel_) {
        return std::u32string(1, *vowel_);
    }
    return U"";
}

std::u32string HangulComposer::current_preedit() const { return compose(); }

}  // namespace hanfe
