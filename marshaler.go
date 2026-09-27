package tmark

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type tmarkTagger interface {
	TmarkTagName() string
}

type tmarkLayouter interface {
	TmarkLayout() ExpandMode
}

type ExpandMode int

const (
	NoExpand ExpandMode = iota
	Expand
	ExpandMultiple
)

type fieldTag struct {
	named bool
	name  string
}

var DefaultMarshaler = Marshaler{}

type Marshaler struct {
	depth int
}

// Marshal serializes a tmark value using struct tags and reflection.
func Marshal(v any) ([]byte, error) {
	return DefaultMarshaler.Marshal(v)
}

// Marshal serializes a tmark value using struct tags and reflection.
func (m Marshaler) Marshal(v any) ([]byte, error) {
	if v == nil {
		return nil, fmt.Errorf("tmark: cannot marshal nil")
	}
	return m.marshalValue(reflect.ValueOf(v))
}

var (
	unknownType = reflect.TypeOf(Unknown{})
	tagNames    = map[reflect.Type]string{
		reflect.TypeOf(Document{}):      "document",
		reflect.TypeOf(AttachedMedia{}): "attached",
		reflect.TypeOf(Caption{}):       "caption",
		reflect.TypeOf(Link{}):          "a",
		reflect.TypeOf(AnchorLink{}):    "anchor-link",
		reflect.TypeOf(Reference{}):     "ref",
		reflect.TypeOf(ReferenceLink{}): "ref-link",
		reflect.TypeOf(Bold{}):          "b",
		reflect.TypeOf(Italic{}):        "i",
		reflect.TypeOf(Marked{}):        "m",
		reflect.TypeOf(Underline{}):     "u",
		reflect.TypeOf(Strikethrough{}): "s",
		reflect.TypeOf(Spoiler{}):       "spoiler",
		reflect.TypeOf(Subscript{}):     "sub",
		reflect.TypeOf(Superscript{}):   "sup",
		reflect.TypeOf(DateTime{}):      "datetime",
		reflect.TypeOf(Code{}):          "code",
		reflect.TypeOf(Math{}):          "math",
		reflect.TypeOf(Icon{}):          "icon",
		reflect.TypeOf(Paragraph{}):     "p",
		reflect.TypeOf(Header{}):        "h",
		reflect.TypeOf(Preformatted{}):  "pre",
		reflect.TypeOf(MathBlock{}):     "math-block",
		reflect.TypeOf(Anchor{}):        "anchor",
		reflect.TypeOf(Divider{}):       "hr",
		reflect.TypeOf(Blockquote{}):    "q",
		reflect.TypeOf(PullQuote{}):     "as",
		reflect.TypeOf(Collage{}):       "collage",
		reflect.TypeOf(List{}):          "list",
		reflect.TypeOf(ListItem{}):      "li",
		reflect.TypeOf(Map{}):           "map",
		reflect.TypeOf(Image{}):         "img",
		reflect.TypeOf(Video{}):         "video",
		reflect.TypeOf(Audio{}):         "audio",
		reflect.TypeOf(Slideshow{}):     "slideshow",
		reflect.TypeOf(Table{}):         "table",
		reflect.TypeOf(TableRow{}):      "tr",
		reflect.TypeOf(Cell{}):          "td",
		reflect.TypeOf(Details{}):       "details",
	}
	expandModes = map[reflect.Type]ExpandMode{
		reflect.TypeOf(Document{}):      Expand,
		reflect.TypeOf(AttachedMedia{}): Expand,
		reflect.TypeOf(Preformatted{}):  Expand,
		reflect.TypeOf(MathBlock{}):     Expand,
		reflect.TypeOf(Collage{}):       Expand,
		reflect.TypeOf(List{}):          Expand,
		reflect.TypeOf(Slideshow{}):     Expand,
		reflect.TypeOf(Table{}):         Expand,
		reflect.TypeOf(Blockquote{}):    ExpandMultiple,
		reflect.TypeOf(ListItem{}):      ExpandMultiple,
		reflect.TypeOf(Details{}):       ExpandMultiple,
	}
)

func escape(s string) []byte {
	n := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '#', '{', '}', '\\':
			n++
		}
	}
	if n == 0 {
		return []byte(s)
	}
	out := make([]byte, len(s)+n)
	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '#', '{', '}', '\\':
			out[j] = '\\'
			j++
		}
		out[j] = c
		j++
	}
	return out
}

func (m *Marshaler) marshalValue(v reflect.Value) ([]byte, error) {
	return m.marshalTypedValue(v, false)
}

func (m *Marshaler) marshalTypedValue(v reflect.Value, omitTag bool) ([]byte, error) {
	if !v.IsValid() {
		return nil, nil
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		return m.marshalTypedValue(v.Elem(), omitTag)
	}
	switch v.Kind() {
	case reflect.Interface:
		if v.IsNil() {
			return nil, nil
		}
		return m.marshalTypedValue(v.Elem(), omitTag)
	case reflect.String:
		if !utf8.ValidString(v.String()) || strings.ContainsRune(v.String(), '\r') {
			return nil, fmt.Errorf("tmark: invalid text")
		}
		return escape(v.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return []byte(strconv.FormatInt(v.Int(), 10)), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return []byte(strconv.FormatUint(v.Uint(), 10)), nil
	case reflect.Float32:
		if math.IsNaN(v.Float()) || math.IsInf(v.Float(), 0) {
			return nil, fmt.Errorf("tmark: non-finite number")
		}
		if v.Float() == 0 {
			return []byte("0"), nil
		}
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 32)), nil
	case reflect.Float64:
		if math.IsNaN(v.Float()) || math.IsInf(v.Float(), 0) {
			return nil, fmt.Errorf("tmark: non-finite number")
		}
		if v.Float() == 0 {
			return []byte("0"), nil
		}
		return []byte(strconv.FormatFloat(v.Float(), 'f', -1, 64)), nil
	case reflect.Bool:
		if v.Bool() {
			return []byte("t"), nil
		}
		return []byte("f"), nil
	case reflect.Slice, reflect.Array:
		return m.marshalSequence(v, false)
	case reflect.Struct:
		if v.Type() == unknownType {
			return m.marshalUnknown(v.Interface().(Unknown))
		}
		return m.marshalStruct(v, omitTag)
	default:
		return nil, fmt.Errorf("tmark: unsupported value type %s", v.Type())
	}
}

func (m *Marshaler) marshalStruct(v reflect.Value, omitTag bool) ([]byte, error) {
	tag, ok := toMarkName(v)
	if !ok {
		return nil, fmt.Errorf("tmark: missing tag name for %s", v.Type())
	}
	if !validIdentifier(tag) {
		return nil, fmt.Errorf("tmark: invalid tag %q", tag)
	}
	out := []byte{'{'}
	if !omitTag {
		out = append(out, tag...)
		out = append(out, ';')
	}
	body, err := m.marshalStructBody(v, expandFor(v), false)
	if err != nil {
		return nil, err
	}
	out = append(out, body...)
	out = append(out, '}')
	return out, nil
}

func (m *Marshaler) marshalStructBody(v reflect.Value, expand ExpandMode, fieldsOnly bool) ([]byte, error) {
	if m.depth >= maxDepth {
		return nil, fmt.Errorf("tmark: max depth %d exceeded", maxDepth)
	}
	m.depth++
	defer func() { m.depth-- }()
	if err := validateNode(v.Interface()); err != nil {
		return nil, err
	}
	var out []byte
	t := v.Type()
	if fieldsOnly && !hasTmarkFields(t) {
		return nil, fmt.Errorf("tmark: missing tag name for %s", v.Type())
	}
	expanded := expand == Expand || expand == ExpandMultiple && primaryLen(v) > 1
	unnamedSeen := false
	for i := 0; i < t.NumField(); i++ {
		ft, ok := parseTmarkTag(t.Field(i))
		if !ok {
			continue
		}
		field := v.Field(i)
		if ft.named {
			if !validIdentifier(ft.name) {
				return nil, fmt.Errorf("tmark: invalid field name %q", ft.name)
			}
			if isEmpty(field) {
				continue
			}
			if expanded {
				out = append(out, '\n')
			}
			var err error
			out, err = m.appendNamed(out, ft.name, field)
			if err != nil {
				return nil, err
			}
		} else {
			if unnamedSeen {
				return nil, fmt.Errorf("tmark: %s has multiple unnamed fields", t.Name())
			}
			unnamedSeen = true
			if isEmpty(field) {
				continue
			}
			if expanded {
				v := indirect(field)
				var b []byte
				var err error
				if v.Kind() == reflect.Slice || v.Kind() == reflect.Array {
					b, err = m.marshalSequence(v, true)
				} else {
					b, err = m.marshalValue(v)
					b = append([]byte{'\n'}, b...)
				}
				if err != nil {
					return nil, err
				}
				out = append(out, b...)
			} else {
				b, err := m.marshalValue(field)
				if err != nil {
					return nil, err
				}
				out = append(out, b...)
			}
		}
	}
	if expanded {
		out = append(out, '\n')
	}
	return out, nil
}

func (m *Marshaler) appendNamed(out []byte, name string, v reflect.Value) ([]byte, error) {
	out = append(out, '#')
	out = append(out, name...)
	out = append(out, '{')
	b, err := m.marshalAttributeValue(v)
	if err != nil {
		return nil, err
	}
	out = append(out, b...)
	out = append(out, '}')
	return out, nil
}

func (m *Marshaler) marshalAttributeValue(v reflect.Value) ([]byte, error) {
	v = indirect(v)
	if !v.IsValid() {
		return nil, nil
	}
	if v.Kind() == reflect.Struct {
		return m.marshalStructBody(v, NoExpand, true)
	}
	return m.marshalValue(v)
}

func (m *Marshaler) marshalSequence(v reflect.Value, expanded bool) ([]byte, error) {
	out := make([]byte, 0)
	concrete := concreteElement(v.Type().Elem())
	for i := 0; i < v.Len(); i++ {
		if expanded {
			out = append(out, '\n')
		}
		b, err := m.marshalTypedValue(v.Index(i), concrete)
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return out, nil
}

func (m *Marshaler) marshalUnknown(u Unknown) ([]byte, error) {
	if len(u.Raw) == 0 {
		return nil, fmt.Errorf("tmark: unknown node raw value is empty")
	}
	parts, err := parseTmark(u.Raw)
	if err != nil {
		return nil, err
	}
	if _, ok := singleNode(parts); !ok {
		return nil, fmt.Errorf("tmark: unknown raw value must be a single node")
	}
	return append([]byte(nil), u.Raw...), nil
}

func hasTmarkFields(t reflect.Type) bool {
	for i := 0; i < t.NumField(); i++ {
		if _, ok := parseTmarkTag(t.Field(i)); ok {
			return true
		}
	}
	return false
}

func primaryLen(v reflect.Value) int {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		ft, ok := parseTmarkTag(t.Field(i))
		if ok && !ft.named {
			return sequenceLen(v.Field(i))
		}
	}
	return 0
}

func concreteElement(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct
}

func parseTmarkTag(sf reflect.StructField) (fieldTag, bool) {
	raw := sf.Tag.Get("tmark")
	if raw == "" || raw == "-" {
		return fieldTag{}, false
	}
	parts := strings.Split(raw, ",")
	ft := fieldTag{name: strings.ToLower(sf.Name)}
	if parts[0] == "named" {
		ft.named = true
	} else if parts[0] == "unnamed" {
		ft.named = false
	} else {
		return fieldTag{}, false
	}
	for _, p := range parts[1:] {
		switch p {
		case "":
		default:
			ft.name = p
		}
	}
	return ft, true
}

func expandFor(v reflect.Value) ExpandMode {
	if mode, ok := expandModes[v.Type()]; ok {
		return mode
	}
	if v.CanInterface() {
		if layouter, ok := v.Interface().(tmarkLayouter); ok {
			return layouter.TmarkLayout()
		}
	}
	return NoExpand
}

func toMarkName(v reflect.Value) (string, bool) {
	if name, ok := tagNames[v.Type()]; ok {
		return name, true
	}
	if v.CanInterface() {
		if tagger, ok := v.Interface().(tmarkTagger); ok {
			return tagger.TmarkTagName(), true
		}
	}
	return "", false
}

func isEmpty(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Array, reflect.Map, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	}
	return false
}

func sequenceLen(v reflect.Value) int {
	v = indirect(v)
	if v.IsValid() && (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) {
		return v.Len()
	}
	return 1
}

func indirect(v reflect.Value) reflect.Value {
	for v.IsValid() && (v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface) {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}
