// Package auth manages authentication, sessions and account profiles.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrInvalidCookies = errors.New("invalid cookies")

type Cookies map[string]string

var requiredRegularCookies = []string{"c_user", "xs"}
var requiredE2EECookies = []string{"c_user", "xs"}

func ParseCookieString(raw string) (Cookies, error) {
	result := Cookies{}
	for _, part := range strings.Split(raw, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			continue
		}
		result[key] = strings.TrimSpace(value)
	}
	if len(result) == 0 {
		return nil, ErrInvalidCookies
	}
	return result, nil
}

type browserCookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func ParseBrowserCookieJSON(data []byte) (Cookies, error) {
	var list []browserCookie
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidCookies, err)
	}
	result := Cookies{}
	for _, cookie := range list {
		if strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		result[cookie.Name] = cookie.Value
	}
	if len(result) == 0 {
		return nil, ErrInvalidCookies
	}
	return result, nil
}

func (c Cookies) ValidateRegular() error { return validateCookies(c, requiredRegularCookies) }
func (c Cookies) ValidateE2EE() error    { return validateCookies(c, requiredE2EECookies) }

func validateCookies(c Cookies, required []string) error {
	missing := make([]string, 0)
	for _, key := range required {
		if strings.TrimSpace(c[key]) == "" {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("%w: missing %s", ErrInvalidCookies, strings.Join(missing, ", "))
	}
	return nil
}

func (c Cookies) String() string {
	keys := make([]string, 0, len(c))
	for key := range c {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var builder strings.Builder
	for index, key := range keys {
		if index > 0 {
			builder.WriteString("; ")
		}
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(c[key])
	}
	return builder.String()
}

func (c Cookies) Clone() Cookies {
	result := make(Cookies, len(c))
	for key, value := range c {
		result[key] = value
	}
	return result
}
