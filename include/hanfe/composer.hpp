#pragma once

#include <optional>
#include <string>
#include <vector>

#include "hanfe/util.hpp"

namespace hanfe {

enum class JamoRole {
    Auto,
    Leading,
    Trailing,
};

struct CompositionResult {
    std::string commit;
    std::string preedit;
};

class HangulComposer {
   public:
    HangulComposer();

    CompositionResult feed(char32_t ch, JamoRole role = JamoRole::Auto);
    std::optional<std::string> backspace();
    std::string flush();

   private:
    std::u32string handle_consonant(char32_t ch, JamoRole role);
    std::u32string handle_vowel(char32_t ch, JamoRole role);
    std::u32string attach_trailing(char32_t ch);

    std::u32string compose() const;
    std::u32string current_preedit() const;

    std::optional<char32_t> leading_;
    std::optional<char32_t> vowel_;
    std::optional<char32_t> trailing_;
};

}  // namespace hanfe
