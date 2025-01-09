package env

import (
	"errors"
	"fmt"
	"iter"
	"os"
	"reflect"
	"strconv"
	"strings"
)

var (
	ErrRequired = errors.New("environment variable is required")
)

// ReadEnv will populate the type with the appropriate environment variables.
func ReadEnv(dst any) error { return ReadEnvPrefixed("", dst) }

// ReadEnvPrefixed is like [ReadEnv] except will use an environment variable
// prefix.
func ReadEnvPrefixed(prefix string, dst any) error {
	prefix = strings.ToUpper(prefix)
	v := reflect.ValueOf(dst)
	return read(prefix, v, nil, 0)
}

const defaultArraySplit = ','

func read(key string, value reflect.Value, t *tag, depth uint) error {
	if value.Kind() != reflect.Pointer {
		return errors.New("cannot populate non pointer type")
	}
	elem := value.Elem()

	switch elem.Kind() {
	case reflect.Invalid:
		return fmt.Errorf("invalid type %v", elem.Kind())
	case reflect.Interface:
		fallthrough // TODO

	case reflect.String:
		raw, ok := os.LookupEnv(key)
		if ok {
			elem.Set(reflect.ValueOf(raw))
		} else if t != nil && t.required {
			return fmt.Errorf("%w: %q", ErrRequired, key)
		}

	case reflect.Slice:
		env, ok := os.LookupEnv(key)
		if ok {
			sep := string(defaultArraySplit)
			if t != nil && len(t.split) > 0 {
				sep = t.split
			}
			strs := strings.Split(env, sep)
			elemType := elem.Type()
			conv := stringConverter(elemType.Elem().Kind()) // inner type kind
			slice := reflect.MakeSlice(
				elemType,
				0,
				len(strs),
			)
			for _, str := range strs {
				s, err := conv(str)
				if err != nil {
					return err
				}
				slice = reflect.Append(slice, s)
			}
			elem.Set(slice)
		} else if t != nil && t.required {
			return fmt.Errorf("%w: %q", ErrRequired, key)
		}

	case reflect.Array:
		return errors.New("arrays are unimplemented")

	case reflect.Struct:
		et := elem.Type()
		n := et.NumField()
		if len(key) > 0 {
			key += "_"
		}
		for i := 0; i < n; i++ {
			sf := et.Field(i)
			name := toSnakeUpper(sf.Name)
			k := key
			if sf.Anonymous {
				name = ""
				if len(k) > 0 {
					k = k[:len(k)-1]
				}
			}

			var (
				t   *tag
				err error
			)
			tag, ok := sf.Tag.Lookup("env")
			if ok && len(tag) > 0 {
				t, err = parseTag(tag)
				if err != nil {
					return err
				}
				if t.skip {
					break // break out of switch
				}
				if len(t.name) > 0 {
					name = t.name
					if t.skipprefix {
						k = ""
					}
				}
			}

			fieldElem := elem.Field(i)
			if sf.Type.Kind() == reflect.Pointer {
				if fieldElem.IsNil() {
					fieldElem.Set(reflect.New(sf.Type.Elem()))
				}
				err = read(k+name, fieldElem, t, depth+1)
			} else {
				err = read(k+name, fieldElem.Addr(), t, depth+1)
			}
			if err != nil {
				return err
			}
		}

	case reflect.Map:
		elemType := elem.Type()
		m := reflect.MakeMap(elemType)
		conv := stringConverter(elemType.Elem().Kind())
		for env, val := range environ(key) {
			v, err := conv(val)
			if err != nil {
				return err
			}
			m.SetMapIndex(reflect.ValueOf(env), v)
		}
		elem.Set(m)

	default:
		conv := stringConverter(elem.Kind())
		e, ok := os.LookupEnv(key)
		if ok {
			parsed, err := conv(e)
			if err != nil {
				return err
			}
			elem.Set(parsed)
		} else if t != nil && t.required {
			return fmt.Errorf("%w: %q", ErrRequired, key)
		}
	}
	return nil
}

type tag struct {
	name       string
	required   bool
	split      string
	skipprefix bool
	skip       bool
}

func parseTag(raw string) (*tag, error) {
	var t tag
	p := strings.Split(raw, ",")
	t.name = p[0]
	if len(p) == 1 {
		return &t, nil
	}
	for _, key := range p[1:] {
		key := strings.ToLower(key)
		val := ""
		if ix := strings.IndexByte(key, '='); ix > 0 {
			val = key[ix+1:]
			key = key[:ix]
		}
		switch key {
		case "required":
			t.required = true
		case "split":
			if len(val) == 0 {
				return nil, errors.New("must specify a split string e.g. split=,")
			}
			t.split = val
		case "skipprefix", "noprefix":
			t.skipprefix = true
		case "-", "skip":
			t.skip = true
		}
	}
	return &t, nil
}

type converter func(s string) (vv reflect.Value, err error)

func stringConverter(kind reflect.Kind) converter {
	switch kind {
	case reflect.String:
		return func(s string) (reflect.Value, error) { return reflect.ValueOf(s), nil }
	case reflect.Int:
		return parseSigned[int]
	case reflect.Int8:
		return parseSigned[int8]
	case reflect.Int16:
		return parseSigned[int16]
	case reflect.Int32:
		return parseSigned[int32]
	case reflect.Int64:
		return parseSigned[int64]
	case reflect.Uint:
		return parseUnsigned[uint]
	case reflect.Uint8:
		return parseUnsigned[uint8]
	case reflect.Uint16:
		return parseUnsigned[uint16]
	case reflect.Uint32:
		return parseUnsigned[uint32]
	case reflect.Uint64:
		return parseUnsigned[uint64]
	case reflect.Float32:
		return parseFloat[float32]
	case reflect.Float64:
		return parseFloat[float64]
	case reflect.Bool:
		return parseBool
	}
	var x reflect.Value
	return func(string) (reflect.Value, error) { return x, errors.New("invalid kind") }
}

type signed interface {
	~int8 | ~int16 | ~int32 | ~int64 | ~int
}

type unsigned interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uint
}

func parseSigned[T signed](s string) (reflect.Value, error) {
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return reflect.ValueOf(T(0)), err
	}
	return reflect.ValueOf(T(v)), nil
}

func parseUnsigned[T unsigned](s string) (reflect.Value, error) {
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return reflect.ValueOf(T(0)), err
	}
	return reflect.ValueOf(T(v)), nil
}

func parseFloat[T ~float32 | ~float64](s string) (reflect.Value, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return reflect.ValueOf(T(0)), err
	}
	return reflect.ValueOf(T(v)), nil
}

func parseBool(s string) (reflect.Value, error) {
	v, err := strconv.ParseBool(s)
	if err != nil {
		return reflect.ValueOf(false), err
	}
	return reflect.ValueOf(v), nil
}

func toSnakeUpper(s string) string {
	return strings.ToUpper(toSnake(s))
}

func toSnake(camel string) (snake string) {
	var b strings.Builder
	diff := 'a' - 'A'
	l := len(camel)
	for i, v := range camel {
		if v >= 'a' {
			b.WriteRune(v)
			continue
		}
		if (i != 0 || i == l-1) && ((i > 0 && rune(camel[i-1]) >= 'a') ||
			(i < l-1 && rune(camel[i+1]) >= 'a')) {
			b.WriteRune('_')
		}
		b.WriteRune(v + diff)
	}
	return b.String()
}

func environ(key ...string) iter.Seq2[string, string] {
	envs := os.Environ()
	if len(key) > 0 && len(key[0]) > 0 {
		prefix := key[0] + "_"
		// filters out by the prefix key passed
		return func(yield func(string, string) bool) {
			var found bool
			for env, val := range environ() {
				env, found = strings.CutPrefix(env, prefix)
				if found && !yield(env, val) {
					return
				}
			}
		}
	} else {
		// does not filter
		return func(yield func(string, string) bool) {
			for _, env := range envs {
				ix := strings.IndexByte(env, '=')
				if ix < 0 {
					continue
				}
				if !yield(env[:ix], env[ix+1:]) {
					return
				}
			}
		}
	}
}
