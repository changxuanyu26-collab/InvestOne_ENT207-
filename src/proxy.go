package main

import (
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type networkConfig struct {
	Mode  string
	proxy *url.URL
}

// An explicit user environment proxy wins. macOS System Settings manual proxy
// is used only when no environment proxy is configured. PAC is not evaluated.
func systemNetworkConfig() networkConfig {
	if runtime.GOOS != "darwin" {
		return networkConfig{Mode: "Environment proxy or direct"}
	}
	out, err := exec.Command("/usr/sbin/scutil", "--proxy").Output()
	if err != nil {
		return networkConfig{Mode: "Could not read macOS proxy settings"}
	}
	cfg := parseSystemProxy(string(out))
	return cfg
}
func parseSystemProxy(s string) networkConfig {
	vals := map[string]string{}
	for _, line := range strings.Split(s, "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), " : ", 2)
		if len(parts) == 2 {
			vals[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	get := func(prefix string, scheme string) (*url.URL, bool) {
		if vals[prefix+"Enable"] != "1" {
			return nil, false
		}
		host := vals[prefix+"Proxy"]
		port := vals[prefix+"Port"]
		n, e := strconv.Atoi(port)
		if host == "" || e != nil || n < 1 || n > 65535 {
			return nil, false
		}
		u, e := url.Parse(scheme + "://" + host + ":" + port)
		return u, e == nil
	}
	if u, ok := get("HTTPS", "http"); ok {
		return networkConfig{Mode: "macOS manual HTTPS proxy", proxy: u}
	}
	if u, ok := get("HTTP", "http"); ok {
		return networkConfig{Mode: "macOS manual HTTP proxy (HTTPS forwarding)", proxy: u}
	}
	if u, ok := get("SOCKS", "socks5"); ok {
		return networkConfig{Mode: "macOS manual SOCKS5 proxy", proxy: u}
	}
	if vals["ProxyAutoConfigEnable"] == "1" || vals["ProxyAutoDiscoveryEnable"] == "1" {
		return networkConfig{Mode: "macOS PAC proxy not resolved automatically; try a manual proxy or HTTPS_PROXY if needed"}
	}
	return networkConfig{Mode: "macOS direct (environment proxy may still apply)"}
}
func configuredTransport() (*http.Transport, string) {
	base := http.DefaultTransport.(*http.Transport).Clone()
	cfg := systemNetworkConfig()
	base.Proxy = func(req *http.Request) (*url.URL, error) {
		u, e := http.ProxyFromEnvironment(req)
		if e != nil || u != nil {
			return u, e
		}
		return cfg.proxy, nil
	}
	base.MaxIdleConns = 20
	base.MaxIdleConnsPerHost = 4
	return base, cfg.Mode
}
