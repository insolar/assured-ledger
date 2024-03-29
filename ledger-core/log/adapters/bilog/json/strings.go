package json

import (
	"encoding/json"
	"math"
	"strconv"
	"unicode/utf8"
)

const hex = "0123456789abcdef"

type escapeMode uint8

const (
	_ escapeMode = iota
	quoteChar
	escapeChar
)

func (v escapeMode) needsEscape() bool {
	return v == escapeChar
}

// nolint:unused
func (v escapeMode) needsQuotes() bool {
	return v >= quoteChar
}

var escapeTable = func() (t [256]escapeMode) {

	for i := 0; i < 0x20; i++ {
		t[i] = escapeChar
	}
	for i := 0x80; i < 0x100; i++ {
		t[i] = escapeChar
	}
	t['\\'] = escapeChar
	t['"'] = escapeChar
	t[' '] = quoteChar
	return
}()

// AppendStrings encodes the input strings to json and
// appends the encoded string list to the input byte slice.
func AppendStrings(dst []byte, vals []string) []byte {
	if len(vals) == 0 {
		return append(dst, '[', ']')
	}
	dst = append(dst, '[')
	dst = AppendString(dst, vals[0])
	if len(vals) > 1 {
		for _, val := range vals[1:] {
			dst = AppendString(append(dst, ','), val)
		}
	}
	dst = append(dst, ']')
	return dst
}

func AppendHex(dst, s []byte) []byte {
	dst = append(dst, '"')
	for _, v := range s {
		dst = append(dst, hex[v>>4], hex[v&0x0f])
	}
	return append(dst, '"')
}

func AppendFloat(dst []byte, val float64, bitSize int) []byte {
	// JSON does not permit NaN or Infinity. A typical JSON encoder would fail
	// with an error, but a logging library wants the data to get thru so we
	// make a tradeoff and store those types as string.
	switch {
	case math.IsNaN(val):
		return append(dst, `"NaN"`...)
	case math.IsInf(val, 1):
		return append(dst, `"+Inf"`...)
	case math.IsInf(val, -1):
		return append(dst, `"-Inf"`...)
	}
	return strconv.AppendFloat(dst, val, 'f', -1, bitSize)
}

func AppendKey(dst []byte, key string) []byte {
	// AppendKey adds comma for an empty slice to allow concatenation of slices
	if n := len(dst); n == 0 || dst[n-1] != '{' {
		dst = append(dst, ',')
	}
	dst = AppendString(dst, key)
	return append(dst, ':')
}

// AppendString encodes the input string to json and appends
// the encoded string to the input byte slice.
//
// The operation loops though each byte in the string looking
// for characters that need json or utf8 encoding. If the string
// does not need encoding, then the string is appended in it's
// entirety to the byte slice.
// If we encounter a byte that does need encoding, switch up
// the operation and perform a byte-by-byte read-encode-append.
func AppendString(dst []byte, s string) []byte {
	// Start with a double quote.
	dst = append(dst, '"')
	// Loop through each character in the string.
	for i := 0; i < len(s); i++ {
		// Check if the character needs encoding. Control characters, slashes,
		// and the double quote need json encoding. Bytes above the ascii
		// boundary needs utf8 encoding.
		if escapeTable[s[i]].needsEscape() {
			// We encountered a character that needs to be encoded. Switch
			// to complex version of the algorithm.
			dst = appendStringComplex(dst, s, i)
			return append(dst, '"')
		}
	}
	// The string has no need for encoding an therefore is directly
	// appended to the byte slice.
	dst = append(dst, s...)
	// End with a double quote
	return append(dst, '"')
}

// AppendOptQuotedString only adds quotes when escaped characters or spaces are present.
func AppendOptQuotedString(dst []byte, s string) []byte {

	var needQuotes = false
	for i := 0; i < len(s); i++ {
		switch escapeTable[s[i]] {
		case escapeChar:
			dst = append(dst, '"')
			// We encountered a character that needs to be encoded. Switch
			// to complex version of the algorithm.
			dst = appendStringComplex(dst, s, i)
			return append(dst, '"')
		case quoteChar:
			needQuotes = true
		}
	}
	if !needQuotes {
		return append(dst, s...)
	}
	dst = append(dst, '"')
	dst = append(dst, s...)
	return append(dst, '"')
}

// appendStringComplex is used by appendString to take over an in
// progress JSON string encoding that encountered a character that needs
// to be encoded.
func appendStringComplex(dst []byte, s string, i int) []byte {
	start := 0
	for i < len(s) {
		b := s[i]
		if b >= utf8.RuneSelf {
			r, size := utf8.DecodeRuneInString(s[i:])
			if r == utf8.RuneError && size == 1 {
				// In case of error, first append previous simple characters to
				// the byte slice if any and append a remplacement character code
				// in place of the invalid sequence.
				if start < i {
					dst = append(dst, s[start:i]...)
				}
				dst = append(dst, `\ufffd`...)
				i += size
				start = i
				continue
			}
			i += size
			continue
		}
		if !escapeTable[b].needsEscape() {
			i++
			continue
		}
		// We encountered a character that needs to be encoded.
		// Let's append the previous simple characters to the byte slice
		// and switch our operation to read and encode the remainder
		// characters byte-by-byte.
		if start < i {
			dst = append(dst, s[start:i]...)
		}
		switch b {
		case '"', '\\':
			dst = append(dst, '\\', b)
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			dst = append(dst, '\\', 'u', '0', '0', hex[b>>4], hex[b&0xF])
		}
		i++
		start = i
	}
	if start < len(s) {
		dst = append(dst, s[start:]...)
	}
	return dst
}

func AppendMarshalled(dst []byte, v interface{}) []byte {
	marshaled, err := json.Marshal(v)
	if err != nil {
		return append(append(append(dst, `"marshaling error: `...), err.Error()...), '"')
	}
	return append(dst, marshaled...)
}
