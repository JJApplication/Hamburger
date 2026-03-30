package dsl_conf

import (
	"Hamburger/internal/constant"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type Option func(*options)

type options struct {
	symbols map[string]any
}

type envRef struct {
	name string
}

type symbolRef struct {
	name string
}

type exprNode struct {
	op    tokenType
	left  any
	right any
	unary bool
}

func WithSymbols(symbols map[string]any) Option {
	return func(o *options) {
		if len(symbols) == 0 {
			return
		}
		if o.symbols == nil {
			o.symbols = map[string]any{}
		}
		for k, v := range symbols {
			o.symbols[k] = v
		}
	}
}

func Unmarshal(data []byte, v any, opts ...Option) error {
	if v == nil {
		return fmt.Errorf("target must not be nil")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}
	p := newParser(string(data))
	root, err := p.parse()
	if err != nil {
		return err
	}
	o := options{
		symbols: map[string]any{
			"AppName":   constant.AppName,
			"Copyright": constant.Copyright,
			"Localhost": constant.Localhost,
			"ZeroHost":  constant.ZeroHost,
			"HTTPPort":  constant.HTTPPort,
			"HTTPSPort": constant.HTTPSPort,
		},
	}
	for _, opt := range opts {
		opt(&o)
	}
	decoded, err := decodeWithType(root, rv.Elem().Type(), &o)
	if err != nil {
		return err
	}
	rv.Elem().Set(decoded)
	return nil
}

func decodeWithType(node any, t reflect.Type, o *options) (reflect.Value, error) {
	if node == nil {
		return reflect.Zero(t), nil
	}
	if t.Kind() == reflect.Ptr {
		elem, err := decodeWithType(node, t.Elem(), o)
		if err != nil {
			return reflect.Value{}, err
		}
		ptr := reflect.New(t.Elem())
		ptr.Elem().Set(elem)
		return ptr, nil
	}
	if t.Kind() == reflect.Interface {
		val, err := decodeInterface(node, o)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(val), nil
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := node.(map[string]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected object for struct %s", t.String())
		}
		out := reflect.New(t).Elem()
		fieldMap := buildFieldMap(t)
		for k, raw := range m {
			idx, exists := fieldMap[k]
			if !exists {
				continue
			}
			field := out.Field(idx)
			if !field.CanSet() {
				continue
			}
			fieldVal, err := decodeWithType(raw, field.Type(), o)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("field %s: %w", t.Field(idx).Name, err)
			}
			field.Set(fieldVal)
		}
		return out, nil
	case reflect.Map:
		m, ok := node.(map[string]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected object for map %s", t.String())
		}
		out := reflect.MakeMapWithSize(t, len(m))
		for k, raw := range m {
			keyVal, err := convertKey(k, t.Key())
			if err != nil {
				return reflect.Value{}, err
			}
			elemVal, err := decodeWithType(raw, t.Elem(), o)
			if err != nil {
				return reflect.Value{}, err
			}
			out.SetMapIndex(keyVal, elemVal)
		}
		return out, nil
	case reflect.Slice:
		items, ok := node.([]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected array for slice %s", t.String())
		}
		out := reflect.MakeSlice(t, 0, len(items))
		for _, item := range items {
			elem, err := decodeWithType(item, t.Elem(), o)
			if err != nil {
				return reflect.Value{}, err
			}
			out = reflect.Append(out, elem)
		}
		return out, nil
	case reflect.Array:
		items, ok := node.([]any)
		if !ok {
			return reflect.Value{}, fmt.Errorf("expected array for array %s", t.String())
		}
		out := reflect.New(t).Elem()
		n := len(items)
		if n > t.Len() {
			n = t.Len()
		}
		for i := 0; i < n; i++ {
			elem, err := decodeWithType(items[i], t.Elem(), o)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(elem)
		}
		return out, nil
	default:
		raw, err := resolveScalar(node, o)
		if err != nil {
			return reflect.Value{}, err
		}
		return convertScalar(raw, t)
	}
}

func buildFieldMap(t reflect.Type) map[string]int {
	out := make(map[string]int, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}
		tag := f.Tag.Get("json")
		if tag != "" {
			name := strings.Split(tag, ",")[0]
			if name != "" && name != "-" {
				out[name] = i
			}
		}
		if _, exists := out[f.Name]; !exists {
			out[f.Name] = i
		}
	}
	return out
}

func convertKey(raw string, t reflect.Type) (reflect.Value, error) {
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(raw).Convert(t), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		v := reflect.New(t).Elem()
		v.SetInt(i)
		return v, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		u, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return reflect.Value{}, err
		}
		v := reflect.New(t).Elem()
		v.SetUint(u)
		return v, nil
	default:
		return reflect.Value{}, fmt.Errorf("unsupported map key type %s", t.String())
	}
}

func decodeInterface(node any, o *options) (any, error) {
	switch val := node.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, item := range val {
			decoded, err := decodeInterface(item, o)
			if err != nil {
				return nil, err
			}
			out[k] = decoded
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(val))
		for _, item := range val {
			decoded, err := decodeInterface(item, o)
			if err != nil {
				return nil, err
			}
			out = append(out, decoded)
		}
		return out, nil
	default:
		return resolveScalar(node, o)
	}
}

func resolveScalar(node any, o *options) (any, error) {
	switch val := node.(type) {
	case envRef:
		raw, exists := os.LookupEnv(val.name)
		if !exists {
			return nil, fmt.Errorf("environment variable %s not found", val.name)
		}
		return raw, nil
	case symbolRef:
		return resolveSymbol(val.name, o)
	case exprNode:
		return evalExpr(val, o)
	default:
		return val, nil
	}
}

func resolveSymbol(name string, o *options) (any, error) {
	up := strings.ToUpper(name)
	switch up {
	case "DATE":
		return time.Now().Format("2006-01-02"), nil
	case "DATETIME":
		return time.Now().Format("2006-01-02 15:04:05"), nil
	case "ARCH":
		return runtime.GOARCH, nil
	case "GOOS":
		return runtime.GOOS, nil
	case "GOVERSION":
		return runtime.Version(), nil
	case "NUMCORE":
		return runtime.NumCPU(), nil
	case "KERNEL":
		return resolveKernelVersion(), nil
	}
	if v, ok := o.symbols[name]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("symbol @%s not found", name)
}

func resolveKernelVersion() string {
	if runtime.GOOS == "linux" {
		data, err := os.ReadFile("/proc/sys/kernel/osrelease")
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	if runtime.GOOS == "windows" {
		out, err := exec.Command("cmd", "/c", "ver").Output()
		if err == nil {
			return strings.TrimSpace(string(out))
		}
	}
	out, err := exec.Command("uname", "-r").Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return runtime.GOOS
}

func evalExpr(n exprNode, o *options) (any, error) {
	if n.unary {
		raw, err := resolveScalar(n.right, o)
		if err != nil {
			return nil, err
		}
		switch n.op {
		case tokenPlus:
			num, err := asNumber(raw)
			if err != nil {
				return nil, err
			}
			return num.any(), nil
		case tokenMinus:
			num, err := asNumber(raw)
			if err != nil {
				return nil, err
			}
			if num.isFloat {
				return -num.f, nil
			}
			return -num.i, nil
		case tokenTilde:
			i, err := asInt(raw)
			if err != nil {
				return nil, err
			}
			return ^i, nil
		default:
			return nil, fmt.Errorf("unsupported unary operator")
		}
	}
	leftRaw, err := resolveScalar(n.left, o)
	if err != nil {
		return nil, err
	}
	rightRaw, err := resolveScalar(n.right, o)
	if err != nil {
		return nil, err
	}
	switch n.op {
	case tokenPlus, tokenMinus, tokenStar, tokenSlash, tokenPercent:
		ln, err := asNumber(leftRaw)
		if err != nil {
			return nil, err
		}
		rn, err := asNumber(rightRaw)
		if err != nil {
			return nil, err
		}
		lf, rf := ln.f, rn.f
		switch n.op {
		case tokenPlus:
			if !ln.isFloat && !rn.isFloat {
				return ln.i + rn.i, nil
			}
			return lf + rf, nil
		case tokenMinus:
			if !ln.isFloat && !rn.isFloat {
				return ln.i - rn.i, nil
			}
			return lf - rf, nil
		case tokenStar:
			if !ln.isFloat && !rn.isFloat {
				return ln.i * rn.i, nil
			}
			return lf * rf, nil
		case tokenSlash:
			if rf == 0 {
				return nil, fmt.Errorf("division by zero")
			}
			if !ln.isFloat && !rn.isFloat && ln.i%rn.i == 0 {
				return ln.i / rn.i, nil
			}
			return lf / rf, nil
		case tokenPercent:
			li, err := asInt(leftRaw)
			if err != nil {
				return nil, err
			}
			ri, err := asInt(rightRaw)
			if err != nil {
				return nil, err
			}
			if ri == 0 {
				return nil, fmt.Errorf("mod by zero")
			}
			return li % ri, nil
		}
	case tokenShiftLeft, tokenShiftRight, tokenAmp, tokenPipe, tokenCaret:
		li, err := asInt(leftRaw)
		if err != nil {
			return nil, err
		}
		ri, err := asInt(rightRaw)
		if err != nil {
			return nil, err
		}
		switch n.op {
		case tokenShiftLeft:
			return li << ri, nil
		case tokenShiftRight:
			return li >> ri, nil
		case tokenAmp:
			return li & ri, nil
		case tokenPipe:
			return li | ri, nil
		case tokenCaret:
			return li ^ ri, nil
		}
	}
	return nil, fmt.Errorf("unsupported binary operator")
}

type number struct {
	isFloat bool
	f       float64
	i       int64
}

func (n number) any() any {
	if n.isFloat {
		return n.f
	}
	return n.i
}

func asNumber(v any) (number, error) {
	switch n := v.(type) {
	case int:
		return number{f: float64(n), i: int64(n)}, nil
	case int8:
		return number{f: float64(n), i: int64(n)}, nil
	case int16:
		return number{f: float64(n), i: int64(n)}, nil
	case int32:
		return number{f: float64(n), i: int64(n)}, nil
	case int64:
		return number{f: float64(n), i: n}, nil
	case uint:
		return number{f: float64(n), i: int64(n)}, nil
	case uint8:
		return number{f: float64(n), i: int64(n)}, nil
	case uint16:
		return number{f: float64(n), i: int64(n)}, nil
	case uint32:
		return number{f: float64(n), i: int64(n)}, nil
	case uint64:
		return number{f: float64(n), i: int64(n)}, nil
	case float32:
		return number{isFloat: true, f: float64(n), i: int64(n)}, nil
	case float64:
		return number{isFloat: true, f: n, i: int64(n)}, nil
	case string:
		if strings.ContainsAny(n, ".eE") {
			f, err := strconv.ParseFloat(n, 64)
			if err != nil {
				return number{}, err
			}
			return number{isFloat: true, f: f, i: int64(f)}, nil
		}
		i, err := strconv.ParseInt(n, 10, 64)
		if err != nil {
			return number{}, err
		}
		return number{f: float64(i), i: i}, nil
	default:
		return number{}, fmt.Errorf("value %v is not number", v)
	}
}

func asInt(v any) (int64, error) {
	n, err := asNumber(v)
	if err != nil {
		return 0, err
	}
	if n.isFloat {
		return int64(n.f), nil
	}
	return n.i, nil
}

func convertScalar(raw any, t reflect.Type) (reflect.Value, error) {
	if raw == nil {
		return reflect.Zero(t), nil
	}
	switch t.Kind() {
	case reflect.String:
		return reflect.ValueOf(fmt.Sprint(raw)).Convert(t), nil
	case reflect.Bool:
		switch v := raw.(type) {
		case bool:
			return reflect.ValueOf(v).Convert(t), nil
		case string:
			b, err := strconv.ParseBool(v)
			if err != nil {
				return reflect.Value{}, err
			}
			return reflect.ValueOf(b).Convert(t), nil
		default:
			return reflect.Value{}, fmt.Errorf("cannot convert %T to bool", raw)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := asInt(raw)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(t).Elem()
		out.SetInt(i)
		return out, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := asInt(raw)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(t).Elem()
		out.SetUint(uint64(i))
		return out, nil
	case reflect.Float32, reflect.Float64:
		n, err := asNumber(raw)
		if err != nil {
			return reflect.Value{}, err
		}
		out := reflect.New(t).Elem()
		out.SetFloat(n.f)
		return out, nil
	default:
		rv := reflect.ValueOf(raw)
		if rv.Type().AssignableTo(t) {
			return rv.Convert(t), nil
		}
		return reflect.Value{}, fmt.Errorf("unsupported scalar conversion from %T to %s", raw, t.String())
	}
}

type tokenType int

const (
	tokenEOF tokenType = iota
	tokenLBrace
	tokenRBrace
	tokenLBracket
	tokenRBracket
	tokenColon
	tokenComma
	tokenString
	tokenNumber
	tokenIdentifier
	tokenEnv
	tokenSymbol
	tokenTrue
	tokenFalse
	tokenNull
	tokenPlus
	tokenMinus
	tokenStar
	tokenSlash
	tokenPercent
	tokenShiftLeft
	tokenShiftRight
	tokenAmp
	tokenPipe
	tokenCaret
	tokenLParen
	tokenRParen
	tokenTilde
)

type token struct {
	typ tokenType
	lit string
	pos int
}

type parser struct {
	src    []rune
	pos    int
	curr   token
	peeked bool
}

func newParser(input string) *parser {
	return &parser{src: []rune(input)}
}

func (p *parser) parse() (any, error) {
	p.next()
	val, err := p.parseValue()
	if err != nil {
		return nil, err
	}
	if p.curr.typ != tokenEOF {
		return nil, p.errf("unexpected token %s", p.curr.lit)
	}
	return val, nil
}

func (p *parser) parseValue() (any, error) {
	switch p.curr.typ {
	case tokenLBrace:
		return p.parseObject()
	case tokenLBracket:
		return p.parseArray()
	case tokenString:
		lit := p.curr.lit
		p.next()
		return lit, nil
	case tokenTrue:
		p.next()
		return true, nil
	case tokenFalse:
		p.next()
		return false, nil
	case tokenNull:
		p.next()
		return nil, nil
	default:
		return p.parseExpr()
	}
}

func (p *parser) parseObject() (any, error) {
	if p.curr.typ != tokenLBrace {
		return nil, p.errf("expected {")
	}
	p.next()
	out := map[string]any{}
	if p.curr.typ == tokenRBrace {
		p.next()
		return out, nil
	}
	for {
		var key string
		switch p.curr.typ {
		case tokenString, tokenIdentifier:
			key = p.curr.lit
		default:
			return nil, p.errf("object key must be string or identifier")
		}
		p.next()
		if p.curr.typ != tokenColon {
			return nil, p.errf("expected : after key %s", key)
		}
		p.next()
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		out[key] = val
		if p.curr.typ == tokenComma {
			p.next()
			if p.curr.typ == tokenRBrace {
				p.next()
				break
			}
			continue
		}
		if p.curr.typ == tokenRBrace {
			p.next()
			break
		}
		return nil, p.errf("expected , or } in object")
	}
	return out, nil
}

func (p *parser) parseArray() (any, error) {
	if p.curr.typ != tokenLBracket {
		return nil, p.errf("expected [")
	}
	p.next()
	out := make([]any, 0)
	if p.curr.typ == tokenRBracket {
		p.next()
		return out, nil
	}
	for {
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		out = append(out, val)
		if p.curr.typ == tokenComma {
			p.next()
			if p.curr.typ == tokenRBracket {
				p.next()
				break
			}
			continue
		}
		if p.curr.typ == tokenRBracket {
			p.next()
			break
		}
		return nil, p.errf("expected , or ] in array")
	}
	return out, nil
}

func (p *parser) parseExpr() (any, error) {
	return p.parseBitOr()
}

func (p *parser) parseBitOr() (any, error) {
	left, err := p.parseBitXor()
	if err != nil {
		return nil, err
	}
	for p.curr.typ == tokenPipe {
		op := p.curr.typ
		p.next()
		right, err := p.parseBitXor()
		if err != nil {
			return nil, err
		}
		left = exprNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseBitXor() (any, error) {
	left, err := p.parseBitAnd()
	if err != nil {
		return nil, err
	}
	for p.curr.typ == tokenCaret {
		op := p.curr.typ
		p.next()
		right, err := p.parseBitAnd()
		if err != nil {
			return nil, err
		}
		left = exprNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseBitAnd() (any, error) {
	left, err := p.parseShift()
	if err != nil {
		return nil, err
	}
	for p.curr.typ == tokenAmp {
		op := p.curr.typ
		p.next()
		right, err := p.parseShift()
		if err != nil {
			return nil, err
		}
		left = exprNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseShift() (any, error) {
	left, err := p.parseAddSub()
	if err != nil {
		return nil, err
	}
	for p.curr.typ == tokenShiftLeft || p.curr.typ == tokenShiftRight {
		op := p.curr.typ
		p.next()
		right, err := p.parseAddSub()
		if err != nil {
			return nil, err
		}
		left = exprNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseAddSub() (any, error) {
	left, err := p.parseMulDiv()
	if err != nil {
		return nil, err
	}
	for p.curr.typ == tokenPlus || p.curr.typ == tokenMinus {
		op := p.curr.typ
		p.next()
		right, err := p.parseMulDiv()
		if err != nil {
			return nil, err
		}
		left = exprNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseMulDiv() (any, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.curr.typ == tokenStar || p.curr.typ == tokenSlash || p.curr.typ == tokenPercent {
		op := p.curr.typ
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		left = exprNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *parser) parseUnary() (any, error) {
	switch p.curr.typ {
	case tokenPlus, tokenMinus, tokenTilde:
		op := p.curr.typ
		p.next()
		right, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return exprNode{op: op, right: right, unary: true}, nil
	default:
		return p.parsePrimary()
	}
}

func (p *parser) parsePrimary() (any, error) {
	switch p.curr.typ {
	case tokenNumber:
		num, err := parseNumberLiteral(p.curr.lit)
		if err != nil {
			return nil, err
		}
		p.next()
		return num, nil
	case tokenEnv:
		ref := envRef{name: p.curr.lit}
		p.next()
		return ref, nil
	case tokenSymbol:
		ref := symbolRef{name: p.curr.lit}
		p.next()
		return ref, nil
	case tokenString:
		s := p.curr.lit
		p.next()
		return s, nil
	case tokenLParen:
		p.next()
		val, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.curr.typ != tokenRParen {
			return nil, p.errf("expected )")
		}
		p.next()
		return val, nil
	default:
		return nil, p.errf("invalid expression token %s", p.curr.lit)
	}
}

func parseNumberLiteral(lit string) (any, error) {
	if strings.ContainsAny(lit, ".eE") {
		f, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			return nil, err
		}
		return f, nil
	}
	i, err := strconv.ParseInt(lit, 10, 64)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func (p *parser) errf(format string, args ...any) error {
	return fmt.Errorf("dsl parse error at %d: %s", p.curr.pos, fmt.Sprintf(format, args...))
}

func (p *parser) next() {
	p.curr = p.scan()
}

func (p *parser) scan() token {
	p.skipWhitespaceAndComments()
	if p.pos >= len(p.src) {
		return token{typ: tokenEOF, pos: p.pos}
	}
	ch := p.src[p.pos]
	start := p.pos
	switch ch {
	case '{':
		p.pos++
		return token{typ: tokenLBrace, lit: "{", pos: start}
	case '}':
		p.pos++
		return token{typ: tokenRBrace, lit: "}", pos: start}
	case '[':
		p.pos++
		return token{typ: tokenLBracket, lit: "[", pos: start}
	case ']':
		p.pos++
		return token{typ: tokenRBracket, lit: "]", pos: start}
	case ':':
		p.pos++
		return token{typ: tokenColon, lit: ":", pos: start}
	case ',':
		p.pos++
		return token{typ: tokenComma, lit: ",", pos: start}
	case '(':
		p.pos++
		return token{typ: tokenLParen, lit: "(", pos: start}
	case ')':
		p.pos++
		return token{typ: tokenRParen, lit: ")", pos: start}
	case '+':
		p.pos++
		return token{typ: tokenPlus, lit: "+", pos: start}
	case '-':
		p.pos++
		return token{typ: tokenMinus, lit: "-", pos: start}
	case '*':
		p.pos++
		return token{typ: tokenStar, lit: "*", pos: start}
	case '/':
		p.pos++
		return token{typ: tokenSlash, lit: "/", pos: start}
	case '%':
		p.pos++
		return token{typ: tokenPercent, lit: "%", pos: start}
	case '&':
		p.pos++
		return token{typ: tokenAmp, lit: "&", pos: start}
	case '|':
		p.pos++
		return token{typ: tokenPipe, lit: "|", pos: start}
	case '^':
		p.pos++
		return token{typ: tokenCaret, lit: "^", pos: start}
	case '~':
		p.pos++
		return token{typ: tokenTilde, lit: "~", pos: start}
	case '<':
		if p.pos+1 < len(p.src) && p.src[p.pos+1] == '<' {
			p.pos += 2
			return token{typ: tokenShiftLeft, lit: "<<", pos: start}
		}
	case '>':
		if p.pos+1 < len(p.src) && p.src[p.pos+1] == '>' {
			p.pos += 2
			return token{typ: tokenShiftRight, lit: ">>", pos: start}
		}
	case '"':
		s, ok := p.scanString()
		if !ok {
			return token{typ: tokenEOF, pos: start, lit: "unterminated string"}
		}
		return token{typ: tokenString, lit: s, pos: start}
	case '$':
		p.pos++
		id := p.scanIdentifier()
		return token{typ: tokenEnv, lit: id, pos: start}
	case '@':
		p.pos++
		id := p.scanIdentifier()
		return token{typ: tokenSymbol, lit: id, pos: start}
	}
	if unicode.IsDigit(ch) {
		return token{typ: tokenNumber, lit: p.scanNumber(), pos: start}
	}
	if isIdentifierStart(ch) {
		id := p.scanIdentifierWithStart()
		switch id {
		case "true":
			return token{typ: tokenTrue, lit: id, pos: start}
		case "false":
			return token{typ: tokenFalse, lit: id, pos: start}
		case "null":
			return token{typ: tokenNull, lit: id, pos: start}
		default:
			return token{typ: tokenIdentifier, lit: id, pos: start}
		}
	}
	p.pos++
	return token{typ: tokenEOF, lit: string(ch), pos: start}
}

func (p *parser) scanString() (string, bool) {
	p.pos++
	start := p.pos
	var b strings.Builder
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if ch == '"' {
			b.WriteString(string(p.src[start:p.pos]))
			p.pos++
			return b.String(), true
		}
		if ch == '\\' {
			b.WriteString(string(p.src[start:p.pos]))
			p.pos++
			if p.pos >= len(p.src) {
				return "", false
			}
			esc := p.src[p.pos]
			switch esc {
			case '"', '\\', '/':
				b.WriteRune(esc)
			case 'n':
				b.WriteRune('\n')
			case 'r':
				b.WriteRune('\r')
			case 't':
				b.WriteRune('\t')
			default:
				b.WriteRune(esc)
			}
			p.pos++
			start = p.pos
			continue
		}
		p.pos++
	}
	return "", false
}

func (p *parser) scanIdentifierWithStart() string {
	start := p.pos
	p.pos++
	for p.pos < len(p.src) && isIdentifierPart(p.src[p.pos]) {
		p.pos++
	}
	return string(p.src[start:p.pos])
}

func (p *parser) scanIdentifier() string {
	start := p.pos
	for p.pos < len(p.src) && isIdentifierPart(p.src[p.pos]) {
		p.pos++
	}
	return string(p.src[start:p.pos])
}

func (p *parser) scanNumber() string {
	start := p.pos
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if unicode.IsDigit(ch) || ch == '.' || ch == 'e' || ch == 'E' || ch == '+' || ch == '-' {
			if (ch == '+' || ch == '-') && p.pos > start {
				prev := p.src[p.pos-1]
				if prev != 'e' && prev != 'E' {
					break
				}
			}
			p.pos++
			continue
		}
		break
	}
	return string(p.src[start:p.pos])
}

func (p *parser) skipWhitespaceAndComments() {
	for p.pos < len(p.src) {
		ch := p.src[p.pos]
		if unicode.IsSpace(ch) {
			p.pos++
			continue
		}
		if ch == '#' {
			for p.pos < len(p.src) && p.src[p.pos] != '\n' {
				p.pos++
			}
			continue
		}
		break
	}
}

func isIdentifierStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentifierPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' || r == '.'
}
