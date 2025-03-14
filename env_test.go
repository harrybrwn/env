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
	type Embedded struct {
		EmbeddedValue string
	}
	type Embedded2 struct {
		YeeYee string
	}
	type Inner struct {
		InnerValue string
	}
	type config struct {
		A string
		*Embedded2
		B struct {
			A    any
			B    int
			Test string `env:"WHAT_IS_THIS,skipprefix"`
		}
		C []string
		D []string `env:",split=:"`
		E map[string]int
		Embedded
		T        *Inner
		Empty    *Inner // should be nil
		S3Bucket string
	}
	setEnvs(map[string]string{
		"X_A":          "a",
		"X_B_B":        "10",
		"X_C":          "one,two,three",
		"X_D":          "a:b:c",
		"X_E_ONE":      "1",
		"X_E_TWO":      "2",
		"X_E_THREE":    "3",
		"X_E_69":       "69",
		"WHAT_IS_THIS": "yeeyeeyee",
		// "X_EMBEDDED_EMBEDDED_VALUE": "em",
		"X_EMBEDDED_VALUE": "em",
		"X_YEE_YEE":        "abc",
		"X_T_INNER_VALUE":  "yes",
		"X_S3_BUCKET":      "test-bucket",
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
	if c.B.Test != "yeeyeeyee" {
		t.Errorf("wrong value for skip prefixed attribute: want %q, got %q", "yeeyeeyee", c.B.Test)
	}
	if c.EmbeddedValue != "em" {
		t.Errorf("wrong value of embedded struct field")
	}
	if c.T == nil {
		t.Fatal("expected nil value to be created")
	}
	if c.T.InnerValue != "yes" {
		t.Errorf("expected %q, got %q", "yes", c.T.InnerValue)
	}
	if c.Embedded2 == nil {
		t.Fatal("embeded struct pointer should not be nil")
	}
	if c.YeeYee != "abc" {
		t.Errorf("expected %q, got %q", "abc", c.YeeYee)
	}
	if c.S3Bucket != "test-bucket" {
		t.Errorf("wrong value for camel case field ending with a number")
	}
	// TODO
	//if c.Empty != nil {
	//	t.Errorf("expected unset nested structure to be nil")
	//}
}

func TestEmbeddedStruct(t *testing.T) {
	type Embedded struct {
		EmbeddedValue string
	}
	type config struct {
		Embedded
	}
	setEnvs(map[string]string{
		"E_EMBEDDED_VALUE": "em",
	})
	var c config
	err := ReadEnvPrefixed("E", &c)
	if err != nil {
		t.Fatal(err)
	}
	if c.EmbeddedValue != "em" {
		t.Errorf("wrong value of embedded struct field: expected %q, got %q", "em", c.EmbeddedValue)
	}
}

func TestNestedNilPointer(t *testing.T) {
	type Inner struct {
		InnerValue string
	}
	type config struct {
		T *Inner
	}
	setEnvs(map[string]string{
		"C_T_INNER_VALUE": "yes",
	})
	var c config
	err := ReadEnvPrefixed("C", &c)
	if err != nil {
		t.Fatal(err)
	}
	if c.T == nil {
		t.Fatal("expected nil value to be created")
	}
	if c.T.InnerValue != "yes" {
		t.Errorf("expected %q, got %q", "yes", c.T.InnerValue)
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
		{"HomeURL", "home_url"},
		{"HomeUrl", "home_url"},
		{"DBLocation", "db_location"},
		{"URLValue", "url_value"},
		{"S3Bucket", "s3_bucket"},
		{"4thInternational", "4th_international"},
		{"Nand2tetrisCourse", "nand2tetris_course"},
		{"Nand2TetrisCourse", "nand2_tetris_course"},
	} {
		if res := toSnake(tt[0]); res != tt[1] {
			t.Errorf("expected snake case of %q to be %q, got %q", tt[0], tt[1], res)
		}
	}
}

func TestGet(t *testing.T) {
	if Get("_NOT_HERE", t.Name()) != t.Name() {
		t.Error("wrong default value")
	}
	if Get("_NOT_HERE") != "" {
		t.Error("wrong empty value")
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
