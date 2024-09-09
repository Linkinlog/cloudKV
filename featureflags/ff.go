package featureflags

import (
	"net/http"
	"os"
)

func init() {
    flagEnabler = map[string]enabler{}
    flagEnabler["tracing"] = hasFlagInCookie
}

var flagEnabler map[string]enabler

type enabler func(flag string, r *http.Request) bool

func Enabled(flag string, r *http.Request) bool {
    if _, ok := os.LookupEnv(flag); ok {
        return true
    }

    e, ok := flagEnabler[flag]
    if !ok {
        return false
    }

    return e(flag, r)
}

func hasFlagInCookie(flag string, r *http.Request) bool {
    _, err := r.Cookie(flag)
    return err == nil
}
