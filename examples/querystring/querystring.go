// Package querystring is the sample system germline governs, small enough to
// read in one sitting and real enough that its boundary is not invented.
//
// A query string is a good example because implementations genuinely
// disagree about it and always have. HTML form submission decodes "+" as a
// space; RFC 3986 says "+" is an ordinary character in a query. Both readings
// ship in real software. That disagreement is not a bug in either one, and
// deciding which of the three closes it belongs to is the whole exercise.
//
// The shape here is the shape the method asks for: an algebra (Query, Parse,
// Render), a port (Codec, with two implementations), and an oracle (the laws
// in laws/ plus the recorded corpus).
package querystring

import (
	"fmt"
	"sort"
	"strings"
)

// A Query is a parsed query string: the keys in the order they were first
// seen, and each key's values in the order they appeared. Order is kept
// because a caller that wants a set can always take one, and a caller that
// needs the order cannot get it back.
type Query struct {
	Keys   []string
	Values map[string][]string
}

// A Codec is one version's reading and writing of a query string. Two
// implementations exist, and every law in laws/ runs against both: that is
// what makes it a conformance suite rather than a test of one of them.
type Codec interface {
	// Name is the version string this codec is replayed under.
	Name() string
	Parse(raw string) (Query, error)
	Render(q Query) string
}

// Add appends a value under a key, remembering first-seen order.
func (q *Query) Add(key, value string) {
	if q.Values == nil {
		q.Values = map[string][]string{}
	}
	if _, seen := q.Values[key]; !seen {
		q.Keys = append(q.Keys, key)
	}
	q.Values[key] = append(q.Values[key], value)
}

// Canonical is a comparable form of a query, for laws that need equality over
// a value containing a map. It sorts keys, so it deliberately forgets order;
// the order law compares Keys directly instead.
func (q Query) Canonical() string {
	keys := append([]string(nil), q.Keys...)
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&sb, "%q=%q;", k, q.Values[k])
	}
	return sb.String()
}

// decode reads percent escapes. plusIsSpace is the one thing the two versions
// disagree about, and it is passed in rather than branched on inside, so that
// the difference lives in exactly one place a reader can find.
func decode(s string, plusIsSpace bool) (string, error) {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] == '+' && plusIsSpace:
			sb.WriteByte(' ')
		case s[i] == '%':
			if i+2 >= len(s) {
				return "", fmt.Errorf("truncated escape at %d in %q", i, s)
			}
			hi, err := hexVal(s[i+1])
			if err != nil {
				return "", err
			}
			lo, err := hexVal(s[i+2])
			if err != nil {
				return "", err
			}
			sb.WriteByte(hi<<4 | lo)
			i += 2
		default:
			sb.WriteByte(s[i])
		}
	}
	return sb.String(), nil
}

func hexVal(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	default:
		return 0, fmt.Errorf("%q is not a hex digit", string(c))
	}
}

// unreserved is every byte that survives encoding untouched, from RFC 3986.
func unreserved(c byte) bool {
	return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' ||
		c == '-' || c == '.' || c == '_' || c == '~'
}

// encode writes percent escapes. spaceAsPlus is the mirror of plusIsSpace.
func encode(s string, spaceAsPlus bool) string {
	var sb strings.Builder
	for i := range len(s) {
		c := s[i]
		switch {
		case unreserved(c):
			sb.WriteByte(c)
		case c == ' ' && spaceAsPlus:
			sb.WriteByte('+')
		default:
			fmt.Fprintf(&sb, "%%%02X", c)
		}
	}
	return sb.String()
}

// parse is the shared reader. Both versions split on & and = the same way;
// only the decoding differs.
func parse(raw string, plusIsSpace bool) (Query, error) {
	var q Query
	if raw == "" {
		return q, nil
	}
	for _, pair := range strings.Split(strings.TrimPrefix(raw, "?"), "&") {
		if pair == "" {
			continue
		}
		rawKey, rawValue, _ := strings.Cut(pair, "=")
		key, err := decode(rawKey, plusIsSpace)
		if err != nil {
			return Query{}, err
		}
		value, err := decode(rawValue, plusIsSpace)
		if err != nil {
			return Query{}, err
		}
		q.Add(key, value)
	}
	return q, nil
}

func render(q Query, spaceAsPlus bool) string {
	pairs := make([]string, 0, len(q.Keys))
	for _, k := range q.Keys {
		for _, v := range q.Values[k] {
			pairs = append(pairs, encode(k, spaceAsPlus)+"="+encode(v, spaceAsPlus))
		}
	}
	return strings.Join(pairs, "&")
}

// FormCodec reads "+" as a space and writes a space as "+", the way an HTML
// form submission does. Version 0.1.0.
type FormCodec struct{}

// Name is the version this codec is replayed under.
func (FormCodec) Name() string { return "0.1.0" }

// Parse reads a query string, decoding "+" as a space.
func (FormCodec) Parse(raw string) (Query, error) { return parse(raw, true) }

// Render writes a query string, encoding a space as "+".
func (FormCodec) Render(q Query) string { return render(q, true) }

// RFCCodec treats "+" as an ordinary character, the way RFC 3986 does, and
// writes a space as "%20". Version 0.2.0.
type RFCCodec struct{}

// Name is the version this codec is replayed under.
func (RFCCodec) Name() string { return "0.2.0" }

// Parse reads a query string, leaving "+" alone.
func (RFCCodec) Parse(raw string) (Query, error) { return parse(raw, false) }

// Render writes a query string, encoding a space as "%20".
func (RFCCodec) Render(q Query) string { return render(q, false) }

// Codecs is every implementation, for a conformance suite to range over.
func Codecs() []Codec { return []Codec{FormCodec{}, RFCCodec{}} }
