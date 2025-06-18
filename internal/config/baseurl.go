package config

import (
	"fmt"
	"net/url"
)

// technically authority also optionally contains userinfo, but ignoring that
func getAuthority(host string, port string) string {
	if len(port) == 0 {
		return host
	}
	return fmt.Sprintf("%s:%s", host, port)
}

func EnvURLOrFatal(prefix string) url.URL {
	scheme := EnvStringOrFatal(prefix + "_SCHEME")
	host := EnvStringOrFatal(prefix + "_HOST")
	port := EnvStringOrFatal(prefix + "_PORT")
	return url.URL{
		Scheme: scheme,
		Host:   getAuthority(host, port),
	}
}

// BaseURL encapsulate settings for scheme, host and port
type BaseURL struct {
	Scheme string
	Host   string
	Port   string
}

// HostPort string for things that don't need a Scheme
func HostPort(key string) string {
	host := EnvStringOrFatal(key + "_HOST")
	port := EnvStringOrFatal(key + "_PORT")
	return getAuthority(host, port)
}

// Listen address with defaults
func ListenAddr(key, defaultHost, defaultPort string) string {
	host := EnvString(key+"_HOST", defaultHost)
	port := EnvString(key+"_PORT", defaultPort)
	return getAuthority(host, port)
}

// BuildURL glue together the pieces of a BaseURL
func (baseURL *BaseURL) BuildURL() string {
	return fmt.Sprintf("%s://%s", baseURL.Scheme, getAuthority(baseURL.Host, baseURL.Port))
}

// EnvBaseURLOrFatal look up env host scheme and port to build a BaseURL
func EnvBaseURLOrFatal(key string) BaseURL {
	return BaseURL{
		Scheme: EnvStringOrFatal(key + "_SCHEME"),
		Host:   EnvStringOrFatal(key + "_HOST"),
		Port:   EnvStringOrFatal(key + "_PORT"),
	}
}

// EnvBaseURLOrDefault look up env host scheme and port to build a BaseURL, or default
func EnvBaseURLOrDefault(key, scheme, host, port string) BaseURL {
	return BaseURL{
		Scheme: EnvString(key+"_SCHEME", scheme),
		Host:   EnvString(key+"_HOST", host),
		Port:   EnvString(key+"_PORT", port),
	}
}
