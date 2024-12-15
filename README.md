# env

Use environment variables as a struct.

```go
type XDG struct {
    CacheHome  string
    State      string `env:"STATE_HOME"`
    RuntimeDir string
    Data       struct {
        Home string
        Dirs []string `env:,split=:"`
    }
    Config struct {
        Home string `env:",required"`
        Dirs []string `env:",split=:"`
    }
}

func main() {
    var xdg XDG
    err := env.ReadEnvPrefixed("xdg", &xdg)
    if err != nil {
        log.Fatal(err)
    }
    if xdg.Data.Home != os.Getenv("XDG_DATA_HOME") {
        log.Fatal("this should be the same")
    }
}
```
