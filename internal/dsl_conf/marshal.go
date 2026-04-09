package dsl_conf

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

// Marshal 将任意对象编码为 hamburger DSL 文本。
func Marshal(v any) ([]byte, error) {
	return marshalWithIndent(v, "", "")
}

// MarshalIndent 将任意对象编码为带缩进的 hamburger DSL 文本。
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	return marshalWithIndent(v, prefix, indent)
}

func marshalWithIndent(v any, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := dslEncoder{
		buf:    &buf,
		prefix: prefix,
		indent: indent,
	}
	if err := enc.encode(reflect.ValueOf(v), 0); err != nil {
		return nil, err
	}
	if indent != "" {
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

type dslEncoder struct {
	buf    *bytes.Buffer
	prefix string
	indent string
}

func (e *dslEncoder) encode(v reflect.Value, level int) error {
	if !v.IsValid() {
		e.buf.WriteString("null")
		return nil
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr {
		if v.IsNil() {
			e.buf.WriteString("null")
			return nil
		}
		return e.encode(v.Elem(), level)
	}

	switch v.Kind() {
	case reflect.Bool:
		if v.Bool() {
			e.buf.WriteString("true")
		} else {
			e.buf.WriteString("false")
		}
	case reflect.String:
		e.buf.WriteString(strconv.Quote(v.String()))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		e.buf.WriteString(strconv.FormatInt(v.Int(), 10))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		e.buf.WriteString(strconv.FormatUint(v.Uint(), 10))
	case reflect.Float32:
		e.buf.WriteString(strconv.FormatFloat(v.Float(), 'f', -1, 32))
	case reflect.Float64:
		e.buf.WriteString(strconv.FormatFloat(v.Float(), 'f', -1, 64))
	case reflect.Map:
		return e.encodeMap(v, level)
	case reflect.Struct:
		return e.encodeStruct(v, level)
	case reflect.Slice, reflect.Array:
		return e.encodeArray(v, level)
	default:
		return fmt.Errorf("unsupported kind: %s", v.Kind().String())
	}
	return nil
}

func (e *dslEncoder) encodeMap(v reflect.Value, level int) error {
	if v.IsNil() {
		e.buf.WriteString("null")
		return nil
	}
	keys := v.MapKeys()
	sort.Slice(keys, func(i, j int) bool {
		return keySortValue(keys[i]) < keySortValue(keys[j])
	})
	return e.encodeObjectFromPairs(level, len(keys), func(i int) (string, reflect.Value, error) {
		keyStr, err := encodeMapKey(keys[i])
		if err != nil {
			return "", reflect.Value{}, err
		}
		return keyStr, v.MapIndex(keys[i]), nil
	})
}

func (e *dslEncoder) encodeStruct(v reflect.Value, level int) error {
	t := v.Type()
	type item struct {
		key string
		val reflect.Value
	}
	fields := make([]item, 0, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name := f.Name
		if tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "" {
				name = tagName
			}
		}
		fields = append(fields, item{key: name, val: v.Field(i)})
	}
	sort.Slice(fields, func(i, j int) bool {
		return fields[i].key < fields[j].key
	})
	return e.encodeObjectFromPairs(level, len(fields), func(i int) (string, reflect.Value, error) {
		return fields[i].key, fields[i].val, nil
	})
}

func (e *dslEncoder) encodeObjectFromPairs(level int, n int, pair func(i int) (string, reflect.Value, error)) error {
	e.buf.WriteByte('{')
	if n == 0 {
		e.buf.WriteByte('}')
		return nil
	}
	if e.indent != "" {
		e.buf.WriteByte('\n')
	}
	for i := range n {
		key, val, err := pair(i)
		if err != nil {
			return err
		}
		if e.indent != "" {
			e.writeIndent(level + 1)
		}
		e.buf.WriteString(formatObjectKey(key))
		e.buf.WriteString(": ")
		if err := e.encode(val, level+1); err != nil {
			return err
		}
		if i < n-1 {
			e.buf.WriteByte(',')
		}
		if e.indent != "" {
			e.buf.WriteByte('\n')
		}
	}
	if e.indent != "" {
		e.writeIndent(level)
	}
	e.buf.WriteByte('}')
	return nil
}

func (e *dslEncoder) encodeArray(v reflect.Value, level int) error {
	e.buf.WriteByte('[')
	n := v.Len()
	if n == 0 {
		e.buf.WriteByte(']')
		return nil
	}
	if e.indent != "" {
		e.buf.WriteByte('\n')
	}
	for i := range n {
		if e.indent != "" {
			e.writeIndent(level + 1)
		}
		if err := e.encode(v.Index(i), level+1); err != nil {
			return err
		}
		if i < n-1 {
			e.buf.WriteByte(',')
		}
		if e.indent != "" {
			e.buf.WriteByte('\n')
		}
	}
	if e.indent != "" {
		e.writeIndent(level)
	}
	e.buf.WriteByte(']')
	return nil
}

func (e *dslEncoder) writeIndent(level int) {
	e.buf.WriteString(e.prefix)
	for range level {
		e.buf.WriteString(e.indent)
	}
}

func keySortValue(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	return fmt.Sprintf("%v", v.Interface())
}

func encodeMapKey(v reflect.Value) (string, error) {
	if !v.IsValid() {
		return "", nil
	}
	switch v.Kind() {
	case reflect.String:
		return v.String(), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10), nil
	default:
		return "", fmt.Errorf("unsupported map key kind: %s", v.Kind().String())
	}
}

func formatObjectKey(key string) string {
	if isBareIdentifier(key) {
		return key
	}
	return strconv.Quote(key)
}

func isBareIdentifier(s string) bool {
	if s == "" {
		return false
	}
	rs := []rune(s)
	if !isIdentifierStartRune(rs[0]) {
		return false
	}
	for _, r := range rs[1:] {
		if !isIdentifierPartRune(r) {
			return false
		}
	}
	return true
}

func isIdentifierStartRune(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentifierPartRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}
