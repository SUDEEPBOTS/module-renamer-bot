package renamer

import "strings"

var StyledBoldSerifMap = map[rune]rune{
	'A': '𝐀', 'B': '𝐁', 'C': '𝐂', 'D': '𝐃', 'E': '𝐄', 'F': '𝐅', 'G': '𝐆',
	'H': '𝐇', 'I': '𝐈', 'J': '𝐉', 'K': '𝐊', 'L': '𝐋', 'M': '𝐌', 'N': '𝐍',
	'O': '𝐎', 'P': '𝐏', 'Q': '𝐐', 'R': '𝐑', 'S': '𝐒', 'T': '𝐓', 'U': '𝐔',
	'V': '𝐕', 'W': '𝐖', 'X': '𝐗', 'Y': '𝐘', 'Z': '𝐙',
	'a': '𝐚', 'b': '𝐛', 'c': '𝐜', 'd': '𝐝', 'e': '𝐞', 'f': '𝐟', 'g': '𝐠',
	'h': '𝐡', 'i': '𝐢', 'j': '𝐣', 'k': '𝐤', 'l': '𝐥', 'm': '𝐦', 'n': '𝐧',
	'o': '𝐨', 'p': '𝐩', 'q': '𝐪', 'r': '𝐫', 's': '𝐬', 't': '𝐭', 'u': '𝐮',
	'v': '𝐯', 'w': '𝐰', 'x': '𝐱', 'y': '𝐲', 'z': '𝐳',
	'0': '𝟎', '1': '𝟏', '2': '𝟐', '3': '𝟑', '4': '𝟒', '5': '𝟓', '6': '𝟔',
	'7': '𝟕', '8': '𝟖', '9': '𝟗',
}

var CustomFancyMap = map[rune]rune{
	'S': '𝐒', 's': 'ѕ',
	'u': 'υ', 'U': '𝐔',
	'p': 'ᴘ', 'P': '𝐏',
	'o': 'σ', 'O': '𝐎',
	'r': 'ꝛ', 'R': '𝐑',
	't': 'ᴛ', 'T': '𝐓',
	'h': '𝐇', 'H': '𝐇',
	'e': 'ᴇ', 'E': '𝐄',
	'l': 'ʟ', 'L': '𝐋',
	'a': 'ᴀ', 'A': '𝐀',
	'd': 'ᴅ', 'D': '𝐃',
	'm': 'ᴍ', 'M': '𝐌',
	'i': 'ɪ', 'I': '𝐈',
	'n': 'ɴ', 'N': '𝐍',
	'b': 'ʙ', 'B': '𝐁',
	'c': 'ᴄ', 'C': '𝐂',
	'k': 'ᴋ', 'K': '𝐊',
	'y': 'ʏ', 'Y': '𝐘',
	'w': 'ᴡ', 'W': '𝐖',
	'x': 'x', 'X': '𝐗',
	'v': 'ᴠ', 'V': '𝐕',
	'g': 'ɢ', 'G': '𝐆',
}

var SmallCapsMap = map[rune]rune{
	'a': 'ᴀ', 'A': 'ᴀ',
	'b': 'ʙ', 'B': 'ʙ',
	'c': 'ᴄ', 'C': 'ᴄ',
	'd': 'ᴅ', 'D': 'ᴅ',
	'e': 'ᴇ', 'E': 'ᴇ',
	'f': 'ғ', 'F': 'ғ',
	'g': 'ɢ', 'G': 'ɢ',
	'h': 'ʜ', 'H': 'ʜ',
	'i': 'ɪ', 'I': 'ɪ',
	'j': 'ᴊ', 'J': 'ᴊ',
	'k': 'ᴋ', 'K': 'ᴋ',
	'l': 'ʟ', 'L': 'ʟ',
	'm': 'ᴍ', 'M': 'ᴍ',
	'n': 'ɴ', 'N': 'ɴ',
	'o': 'ᴏ', 'O': 'ᴏ',
	'p': 'ᴘ', 'P': 'ᴘ',
	'q': 'ǫ', 'Q': 'ǫ',
	'r': 'ʀ', 'R': 'ʀ',
	's': 's', 'S': 's',
	't': 'ᴛ', 'T': 'ᴛ',
	'u': 'ᴜ', 'U': 'ᴜ',
	'v': 'ᴠ', 'V': 'ᴠ',
	'w': 'ᴡ', 'W': 'ᴡ',
	'x': 'x', 'X': 'x',
	'y': 'ʏ', 'Y': 'ʏ',
	'z': 'ᴢ', 'Z': 'ᴢ',
}

func ToBoldSerif(text string) string {
	var sb strings.Builder
	for _, r := range text {
		if mapped, ok := StyledBoldSerifMap[r]; ok {
			sb.WriteRune(mapped)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func ToAestheticFancy(text string) string {
	var sb strings.Builder
	for _, r := range text {
		if mapped, ok := CustomFancyMap[r]; ok {
			sb.WriteRune(mapped)
		} else if mapped, ok := StyledBoldSerifMap[r]; ok {
			sb.WriteRune(mapped)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func ToSmallCaps(text string) string {
	var sb strings.Builder
	for _, r := range text {
		if mapped, ok := SmallCapsMap[r]; ok {
			sb.WriteRune(mapped)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}
