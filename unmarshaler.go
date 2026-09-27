package tmark

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

var DefaultUnmarshaler = Unmarshaler{}

type Unmarshaler struct {
	soft bool
}

func (u Unmarshaler) Soft() Unmarshaler {
	u.soft = true
	return u
}

// Unmarshal deserializes a tmark value into a non-nil pointer.
func Unmarshal(data []byte, v any) error {
	return DefaultUnmarshaler.Unmarshal(data, v)
}

// Unmarshal deserializes a tmark value into a non-nil pointer.
func (u Unmarshaler) Unmarshal(data []byte, v any) error {
	if v == nil {
		return fmt.Errorf("tmark: cannot unmarshal into nil")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return fmt.Errorf("tmark: unmarshal target must be a non-nil pointer")
	}
	parts, err := parseTmark(data)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return fmt.Errorf("tmark: empty input")
	}
	return u.assignParts(parts, rv.Elem(), false)
}

type parsedPart struct {
	text  string
	node  *parsedNode
	field *parsedField
}

type parsedNode struct {
	tag  string
	body []parsedPart
	raw  []byte
}

type parsedField struct {
	name  string
	value []parsedPart
}

type tmarkParser struct {
	data  []byte
	pos   int
	depth int
}

func parseTmark(data []byte) ([]parsedPart, error) {
	if !utf8.Valid(data) || bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) {
		return nil, fmt.Errorf("tmark: input must be UTF-8 without BOM")
	}
	for i := range data {
		if data[i] == '\r' && (i+1 == len(data) || data[i+1] != '\n') {
			return nil, fmt.Errorf("tmark: standalone CR at byte %d", i)
		}
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	p := tmarkParser{data: data}
	parts, err := p.parseParts(false)
	if err != nil {
		return nil, err
	}
	if p.pos != len(data) {
		return nil, fmt.Errorf("tmark: unexpected input at byte %d", p.pos)
	}
	return parts, nil
}

func (p *tmarkParser) parseParts(stopOnBrace bool) ([]parsedPart, error) {
	var parts []parsedPart
	seenFields := map[string]bool{}
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case '}':
			if stopOnBrace {
				p.pos++
				return parts, nil
			}
			return nil, fmt.Errorf("tmark: unexpected } at byte %d", p.pos)
		case '{':
			node, err := p.parseNode()
			if err != nil {
				return nil, err
			}
			parts = append(parts, parsedPart{node: node})
		case '#':
			field, err := p.parseField()
			if err != nil {
				return nil, err
			}
			if seenFields[field.name] {
				return nil, fmt.Errorf("tmark: duplicate field %q", field.name)
			}
			seenFields[field.name] = true
			parts = append(parts, parsedPart{field: field})
		default:
			text, err := p.parseText()
			if err != nil {
				return nil, err
			}
			if text != "" {
				parts = append(parts, parsedPart{text: text})
			}
		}
	}
	if stopOnBrace {
		return nil, fmt.Errorf("tmark: missing }")
	}
	return parts, nil
}

func (p *tmarkParser) parseNode() (*parsedNode, error) {
	rawStart := p.pos
	p.depth++
	if p.depth > maxDepth {
		return nil, fmt.Errorf("tmark: max depth %d exceeded", maxDepth)
	}
	defer func() { p.depth-- }()

	p.pos++
	start := p.pos
	for p.pos < len(p.data) && !strings.ContainsRune(";}{#\\", rune(p.data[p.pos])) {
		p.pos++
	}
	if p.pos < len(p.data) && p.data[p.pos] == ';' {
		tag := string(p.data[start:p.pos])
		if !validIdentifier(tag) {
			return nil, fmt.Errorf("tmark: invalid tag %q", tag)
		}
		p.pos++
		body, err := p.parseParts(true)
		if err != nil {
			return nil, err
		}
		return &parsedNode{tag: tag, body: body, raw: p.data[rawStart:p.pos]}, nil
	}
	p.pos = start
	body, err := p.parseParts(true)
	if err != nil {
		return nil, err
	}
	return &parsedNode{body: body, raw: p.data[rawStart:p.pos]}, nil
}

func (p *tmarkParser) parseField() (*parsedField, error) {
	p.pos++
	start := p.pos
	for p.pos < len(p.data) && isFieldNameChar(p.data[p.pos]) {
		p.pos++
	}
	if start == p.pos || p.pos >= len(p.data) || p.data[p.pos] != '{' {
		return nil, fmt.Errorf("tmark: malformed field at byte %d", start-1)
	}
	name := string(p.data[start:p.pos])
	if !validIdentifier(name) {
		return nil, fmt.Errorf("tmark: invalid field name %q", name)
	}
	p.pos++
	value, err := p.parseParts(true)
	if err != nil {
		return nil, err
	}
	return &parsedField{name: name, value: value}, nil
}

func (p *tmarkParser) parseText() (string, error) {
	var b strings.Builder
	for p.pos < len(p.data) {
		c := p.data[p.pos]
		if c == '{' || c == '}' || c == '#' {
			break
		}
		if c == '\\' {
			p.pos++
			if p.pos >= len(p.data) {
				return "", fmt.Errorf("tmark: trailing escape")
			}
			c = p.data[p.pos]
			if c != '#' && c != '{' && c != '}' && c != '\\' {
				return "", fmt.Errorf("tmark: invalid escape at byte %d", p.pos-1)
			}
		}
		b.WriteByte(c)
		p.pos++
	}
	return b.String(), nil
}

func isFieldNameChar(c byte) bool {
	return c == '_' || c == '-' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z'
}

func validIdentifier(s string) bool {
	if len(s) < 1 || len(s) > 32 {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isFieldNameChar(s[i]) {
			return false
		}
	}
	return true
}

func (u Unmarshaler) assignParts(parts []parsedPart, dst reflect.Value, expanded bool) error {
	dst = indirectAlloc(dst)
	if !dst.IsValid() {
		return nil
	}
	switch dst.Kind() {
	case reflect.Interface:
		return u.assignInterface(parts, dst, expanded)
	case reflect.Slice:
		return u.assignSlice(parts, dst, expanded)
	case reflect.Array:
		return u.assignArray(parts, dst)
	case reflect.Struct:
		if dst.Type() == unknownType {
			node, ok := singleNode(parts)
			if !ok {
				return fmt.Errorf("tmark: expected unknown node")
			}
			unknown, err := u.unknownFromNode(node)
			if err != nil {
				return err
			}
			dst.Set(reflect.ValueOf(unknown))
			return nil
		}
		node, ok := singleNode(parts)
		if !ok {
			return fmt.Errorf("tmark: expected %s node", dst.Type())
		}
		return u.assignStruct(node, dst)
	default:
		return assignScalar(parts, dst, expanded)
	}
}

func assignScalar(parts []parsedPart, dst reflect.Value, expanded bool) error {
	value, err := partsText(parts, expanded)
	if err != nil {
		return err
	}
	switch dst.Kind() {
	case reflect.String:
		dst.SetString(value)
	case reflect.Bool:
		v, err := parseBool(value)
		if err != nil {
			return err
		}
		dst.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr,
		reflect.Float32, reflect.Float64:
		if !validNumber(value) {
			return fmt.Errorf("tmark: non-canonical number %q", value)
		}
		switch dst.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if strings.ContainsRune(value, '.') {
				return fmt.Errorf("tmark: expected integer, got %q", value)
			}
			v, err := strconv.ParseInt(value, 10, dst.Type().Bits())
			if err != nil {
				return err
			}
			dst.SetInt(v)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			if strings.ContainsRune(value, '.') {
				return fmt.Errorf("tmark: expected integer, got %q", value)
			}
			v, err := strconv.ParseUint(value, 10, dst.Type().Bits())
			if err != nil {
				return err
			}
			dst.SetUint(v)
		default:
			v, err := strconv.ParseFloat(value, dst.Type().Bits())
			if err != nil {
				return err
			}
			dst.SetFloat(v)
		}
	default:
		return fmt.Errorf("tmark: unsupported target type %s", dst.Type())
	}
	return nil
}

func (u Unmarshaler) assignInterface(parts []parsedPart, dst reflect.Value, expanded bool) error {
	if expanded {
		parts = dropStructuralNewlines(parts)
	}
	if len(parts) != 1 {
		return fmt.Errorf("tmark: expected single value for %s", dst.Type())
	}
	if parts[0].node == nil {
		text := reflect.ValueOf(Text(parts[0].text))
		if text.Type().AssignableTo(dst.Type()) {
			dst.Set(text)
			return nil
		}
		return fmt.Errorf("tmark: text is not assignable to %s", dst.Type())
	}
	t, ok := typeForTag(parts[0].node.tag)
	if !ok {
		if u.soft {
			unknown, err := u.unknownFromNode(parts[0].node)
			if err != nil {
				return err
			}
			v := reflect.ValueOf(unknown)
			if v.Type().AssignableTo(dst.Type()) {
				dst.Set(v)
				return nil
			}
			return fmt.Errorf("tmark: %s is not assignable to %s", v.Type(), dst.Type())
		}
		return fmt.Errorf("tmark: unknown tag %q", parts[0].node.tag)
	}
	v := reflect.New(t).Elem()
	if err := u.assignStruct(parts[0].node, v); err != nil {
		return err
	}
	if !v.Type().AssignableTo(dst.Type()) {
		return fmt.Errorf("tmark: %s is not assignable to %s", v.Type(), dst.Type())
	}
	dst.Set(v)
	return nil
}

func (u Unmarshaler) assignSlice(parts []parsedPart, dst reflect.Value, expanded bool) error {
	elemType := dst.Type().Elem()
	out := reflect.MakeSlice(dst.Type(), 0, len(parts))
	for _, part := range parts {
		if expanded && isStructuralNewline(part) {
			continue
		}
		item := reflect.New(elemType).Elem()
		if concreteElement(elemType) {
			if part.node == nil {
				return fmt.Errorf("tmark: expected concrete array element for %s", dst.Type())
			}
			node := part.node
			body, err := parseTmark(node.raw[1 : len(node.raw)-1])
			if err != nil {
				return err
			}
			if err := u.assignStructBody(body, item, false); err != nil {
				return err
			}
			out = reflect.Append(out, item)
			continue
		}
		if err := u.assignParts([]parsedPart{part}, item, false); err != nil {
			return err
		}
		out = reflect.Append(out, item)
	}
	dst.Set(out)
	return nil
}

func (u Unmarshaler) assignArray(parts []parsedPart, dst reflect.Value) error {
	parts = dropStructuralNewlines(parts)
	if len(parts) != dst.Len() {
		return fmt.Errorf("tmark: array length mismatch: got %d, want %d", len(parts), dst.Len())
	}
	for i, part := range parts {
		if err := u.assignParts([]parsedPart{part}, dst.Index(i), false); err != nil {
			return err
		}
	}
	return nil
}

func (u Unmarshaler) assignStruct(node *parsedNode, dst reflect.Value) error {
	want, ok := toMarkName(dst)
	if !ok || node.tag != want {
		return fmt.Errorf("tmark: expected tag %q for %s, got %q", want, dst.Type(), node.tag)
	}
	return u.assignStructBody(node.body, dst, false)
}

func (u Unmarshaler) assignStructBody(parts []parsedPart, dst reflect.Value, fieldsOnly bool) error {
	t := dst.Type()
	if fieldsOnly && !hasTmarkFields(t) {
		return fmt.Errorf("tmark: missing tag name for %s", t)
	}
	expanded := !fieldsOnly && expandFor(dst) != NoExpand
	unnamedSequence := false
	for i := 0; i < t.NumField(); i++ {
		if ft, ok := parseTmarkTag(t.Field(i)); ok && !ft.named {
			kind := t.Field(i).Type.Kind()
			unnamedSequence = kind == reflect.Slice || kind == reflect.Array
		}
	}
	if expanded && unnamedSequence {
		parts = trimExpandedBody(parts)
	}
	lastNamed := -1
	for i, part := range parts {
		if part.field != nil {
			lastNamed = i
		}
	}
	fields := map[string][]parsedPart{}
	var unnamed []parsedPart
	unnamedStarted := false
	for i, part := range parts {
		if expanded && isStructuralNewline(part) && (unnamedSequence || i < lastNamed) {
			continue
		}
		if part.field != nil {
			if unnamedStarted {
				return fmt.Errorf("tmark: named field follows unnamed value")
			}
			fields[part.field.name] = part.field.value
			continue
		}
		unnamed = append(unnamed, part)
		unnamedStarted = true
	}
	unnamedUsed := false
	for i := 0; i < t.NumField(); i++ {
		ft, ok := parseTmarkTag(t.Field(i))
		if !ok {
			continue
		}
		if ft.named {
			value, ok := fields[ft.name]
			if !ok {
				continue
			}
			delete(fields, ft.name)
			field := indirectAlloc(dst.Field(i))
			if err := u.assignFieldValue(value, field); err != nil {
				return fmt.Errorf("tmark: field %s.%s: %w", t, t.Field(i).Name, err)
			}
			continue
		}
		if len(unnamed) == 0 {
			continue
		}
		unnamedUsed = true
		field := indirectAlloc(dst.Field(i))
		if err := u.assignParts(unnamed, field, expanded); err != nil {
			return fmt.Errorf("tmark: field %s.%s: %w", t, t.Field(i).Name, err)
		}
	}
	if len(fields) > 0 {
		for name := range fields {
			return fmt.Errorf("tmark: unknown field %q for %s", name, t)
		}
	}
	if !unnamedUsed && len(dropStructuralNewlines(unnamed)) > 0 {
		return fmt.Errorf("tmark: unexpected unnamed value for %s", t)
	}
	return validateNode(dst.Interface())
}

func (u Unmarshaler) assignFieldValue(parts []parsedPart, dst reflect.Value) error {
	dst = indirectAlloc(dst)
	if dst.IsValid() && dst.Kind() == reflect.Struct {
		return u.assignStructBody(parts, dst, true)
	}
	return u.assignParts(parts, dst, false)
}

func (u Unmarshaler) unknownFromNode(node *parsedNode) (Unknown, error) {
	if node.tag == "" {
		return Unknown{}, fmt.Errorf("tmark: expected unknown node tag")
	}
	raw := append([]byte(nil), node.raw...)
	return Unknown{Raw: raw}, nil
}

func indirectAlloc(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func singleNode(parts []parsedPart) (*parsedNode, bool) {
	parts = dropStructuralNewlines(parts)
	if len(parts) != 1 || parts[0].node == nil {
		return nil, false
	}
	return parts[0].node, true
}

func partsText(parts []parsedPart, expanded bool) (string, error) {
	var b strings.Builder
	for _, part := range parts {
		if part.node != nil || part.field != nil {
			return "", fmt.Errorf("tmark: expected text")
		}
		b.WriteString(part.text)
	}
	text := b.String()
	if expanded {
		text = strings.TrimPrefix(text, "\n")
		text = strings.TrimSuffix(text, "\n")
	}
	return text, nil
}

func parseBool(s string) (bool, error) {
	switch s {
	case "t":
		return true, nil
	case "f":
		return false, nil
	default:
		return false, fmt.Errorf("tmark: parse bool: expected t or f, got %q", s)
	}
}

func isStructuralNewline(part parsedPart) bool {
	return part.node == nil && part.field == nil && strings.Trim(part.text, "\n") == ""
}

func dropStructuralNewlines(parts []parsedPart) []parsedPart {
	out := parts[:0]
	for _, part := range parts {
		if !isStructuralNewline(part) {
			out = append(out, part)
		}
	}
	return out
}

func trimExpandedBody(parts []parsedPart) []parsedPart {
	for len(parts) > 0 && isStructuralNewline(parts[0]) {
		parts = parts[1:]
	}
	for len(parts) > 0 && isStructuralNewline(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func typeForTag(tag string) (reflect.Type, bool) {
	for typ, name := range tagNames {
		if name == tag {
			return typ, true
		}
	}
	return nil, false
}
