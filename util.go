package env

import "os"

func Get(key string, defaultValue ...string) string {
	v, ok := os.LookupEnv(key)
	if !ok && len(defaultValue) > 0 {
		return defaultValue[0]
	}
	return v
}
