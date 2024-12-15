package env

import (
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestReadEnv(t *testing.T) {
	type config struct {
		A string
		B struct {
			A any
			B int
		}
		C []string
		D []string `env:",split=:"`
		E map[string]int
	}
	setEnvs(map[string]string{
		"X_A":       "a",
		"X_B_B":     "10",
		"X_C":       "one,two,three",
		"X_D":       "a:b:c",
		"X_E_ONE":   "1",
		"X_E_TWO":   "2",
		"X_E_THREE": "3",
		"X_E_69":    "69",
	})
	var c config
	err := ReadEnvPrefixed("X", &c)
	if err != nil {
		t.Fatal(err)
	}
	if c.A != "a" {
		t.Error("wrong value")
	}
	if c.B.B != 10 {
		t.Error("wrong value")
	}
	if !slices.Equal(c.C, []string{"one", "two", "three"}) {
		t.Error("wrong value")
	}
	if !slices.Equal(c.D, []string{"a", "b", "c"}) {
		t.Error("wrong value")
	}
	if !maps.Equal(c.E, map[string]int{"ONE": 1, "TWO": 2, "THREE": 3, "69": 69}) {
		t.Error("wrong value")
	}
}

func TestReadEnv_Err(t *testing.T) {
	err := ReadEnv(0)
	if err == nil {
		t.Fatal("expected an error")
	}
	type C struct {
		X string `env:",split"`
	}
	err = ReadEnv(&C{})
	if err == nil {
		t.Fatal("expected an error")
	}
	type B struct {
		X string `env:"__X__,required"`
	}
	err = ReadEnv(&B{})
	if err == nil {
		t.Fatal("expected an error")
	}
	type A struct {
		X uint16 `env:"__A__X__,required"`
	}
	err = ReadEnv(&A{})
	if err == nil {
		t.Fatal("expected an error")
	}
	os.Setenv("__A__X__", "butts")
	err = ReadEnv(&A{})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestMap(t *testing.T) {
	os.Setenv("XDG_CACHE_HOME", "x")
	os.Setenv("XDG_CONFIG_DIRS", "/usr:/usr/local:/usr/share/")
	os.Setenv("XDG_CONFIG_NUMS", "5,4,3,2,1")
	var c map[string]string
	err := ReadEnvPrefixed("xdg", &c)
	if err != nil {
		t.Fatal(err)
	}
	if c["CONFIG_DIRS"] != "/usr:/usr/local:/usr/share/" {
		t.Error("wrong value")
	}
	if c["CACHE_HOME"] != "x" {
		t.Error("wrong value")
	}
	if c["CONFIG_NUMS"] != "5,4,3,2,1" {
		t.Error("wrong value")
	}
	clear(c)
	err = ReadEnv(&c)
	if err != nil {
		t.Fatal(err)
	}
	if c["XDG_CONFIG_DIRS"] != "/usr:/usr/local:/usr/share/" {
		t.Error("wrong value")
	}
	if c["XDG_CACHE_HOME"] != "x" {
		t.Error("wrong value")
	}
	if c["XDG_CONFIG_NUMS"] != "5,4,3,2,1" {
		t.Error("wrong value")
	}
}

func TestXDG(t *testing.T) {
	type config struct {
		CacheHome string
		DataHome  string
		State     string `env:"STATE_HOME"`
		Config    struct {
			Dirs []string `env:",split=:"`
			Home string
			Nums []int
		} `env:"CONFIG"`
		Session struct {
			Class   string
			Desktop string
			Type    string
		}
	}
	os.Setenv("XDG_CACHE_HOME", "x")
	os.Setenv("XDG_CONFIG_DIRS", "/usr/share/ubuntu:/usr/local/share/:/usr/share/:/var/lib/snapd/desktop")
	os.Setenv("XDG_CONFIG_NUMS", "5,4,3,2,1")
	os.Setenv("XDG_STATE_HOME", "testing123")
	var c config
	err := ReadEnvPrefixed("xdg", &c)
	if err != nil {
		t.Fatal(err)
	}
	if c.State != "testing123" {
		t.Error("wrong state, explicit env should be STATE_HOME")
	}
	if c.CacheHome != "x" {
		t.Error("failed to normalize field names to snake case")
	}
	if !slices.Equal(c.Config.Nums, []int{5, 4, 3, 2, 1}) {
		t.Error("wrong int slice")
	}
	if !slices.Equal(
		c.Config.Dirs,
		[]string{
			"/usr/share/ubuntu",
			"/usr/local/share/",
			"/usr/share/",
			"/var/lib/snapd/desktop",
		},
	) {
		t.Error("wrong xdg_config_dirs")
	}
}

func TestParseTag(t *testing.T) {
	tag, err := parseTag(",split=:")
	if err != nil {
		t.Fatal(err)
	}
	if tag.split != ":" {
		t.Error("expected \":\"")
	}
	tag = must(parseTag(",required"))
	if !tag.required {
		t.Fatal("expected required")
	}
	tag = must(parseTag(",split=|,required"))
	if tag.split != "|" {
		t.Fatal("expected split to be \"|\"")
	}
	if !tag.required {
		t.Fatal("expected required")
	}
	tag = must(parseTag(",required,split=|"))
	if tag.split != "|" {
		t.Fatal("expected split to be \"|\"")
	}
	if !tag.required {
		t.Fatal("expected required")
	}
}

func TestParsers(t *testing.T) {
	type table struct {
		in   string
		kind reflect.Kind
		exp  any
	}
	for _, tt := range []table{
		{"1.1", reflect.Float32, float32(1.1)},
		{"1.2", reflect.Float64, 1.2},
		{"true", reflect.Bool, true},
		{"false", reflect.Bool, false},
		{"1", reflect.Bool, true},
		{"9", reflect.Int8, int8(9)},
		{"9", reflect.Int16, int16(9)},
		{"9", reflect.Int32, int32(9)},
		{"9", reflect.Int64, int64(9)},
		{"9", reflect.Int, int(9)},
		{"9", reflect.Uint8, uint8(9)},
		{"9", reflect.Uint16, uint16(9)},
		{"9", reflect.Uint32, uint32(9)},
		{"9", reflect.Uint64, uint64(9)},
		{"9", reflect.Uint, uint(9)},
	} {
		conv := stringConverter(tt.kind)
		res, err := conv(tt.in)
		if err != nil {
			t.Error(err)
			continue
		}
		exp := reflect.ValueOf(tt.exp)
		if !res.Equal(exp) {
			t.Errorf("expected %#v to equal %#v", res, exp)
		}
	}

	for _, tt := range []table{
		{"1.0", reflect.Int, nil},
		{"-1", reflect.Uint, nil},
		{"hello", reflect.Float64, nil},
		{"", reflect.Invalid, nil},
		{"", reflect.Bool, nil},
		{"99999999999999999999", reflect.Uint8, nil},
	} {
		conv := stringConverter(tt.kind)
		_, err := conv(tt.in)
		if err == nil {
			t.Errorf("expected an error from %#v", tt)
			continue
		}
	}
}

func TestToSnake(t *testing.T) {
	for _, tt := range [][2]string{
		{"toSnake", "to_snake"},
		{"ToSnake", "to_snake"},
		{"value", "value"},
		{"vaLue", "va_lue"},
	} {
		if res := toSnake(tt[0]); res != tt[1] {
			t.Errorf("expected snake case of %q to be %q, got %q", tt[0], tt[1], res)
		}
	}
}

func setEnvs(m map[string]string) {
	for k, v := range m {
		os.Setenv(strings.ToUpper(k), v)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
