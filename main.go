package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

var (
	hostRemote string = "localhost"
	hostLocal  string = "localhost"
	portLocal  uint   = 8080
	portRemote uint   = 80
)

func init() {
	flag.StringVar(&hostRemote, "host-remote", hostRemote, "remote host")
	flag.StringVar(&hostLocal, "host-local", hostLocal, "local host")
	flag.UintVar(&portLocal, "port-local", portLocal, "local port")
	flag.UintVar(&portRemote, "port-remote", portRemote, "remote port")
}

func init() {
	if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		var lvl slog.Level
		// UnmarshalText can parse level strings like "DEBUG", "INFO", "WARN", "ERROR"
		if err := lvl.UnmarshalText([]byte(strings.ToLower(envLevel))); err == nil {
			slog.SetLogLoggerLevel(lvl)
		} else {
			// Handle the error if the environment variable value is invalid
			slog.Default().Error("invalid LOG_LEVEL environment variable, using default INFO", "err", err, "invalid_level", envLevel)
		}
	}

}

const (
	proxyHeader = "X-Proxy"
	proxyValue  = "Once"
)

func main() {
	flag.Parse()
	from := fmt.Sprintf("%s:%d", hostLocal, portLocal)
	rewrite := fmt.Sprintf("%s:%d", hostRemote, portRemote)
	handler := func(p *httputil.ReverseProxy) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(proxyHeader) == proxyValue {
				slog.Info("Skipping proxying request again", "url", r.URL)
				return
			}
			log.Println(r.URL)
			r.Host = rewrite
			// oldPath := r.URL.Path
			// r.URL.Path = strings.Replace(r.URL.Path, from, rewrite, 1)
			// log.Printf("Rewriting request path from %s to %s\n", oldPath, r.URL.Path)
			slog.Debug("Proxying request", "url", r.URL, "host", r.Host)
			// r.URL.Port = portRemote
			w.Header().Set(proxyHeader, proxyValue)
			p.ServeHTTP(w, r)
		}
	}

	remote := &url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", hostRemote, portRemote),
	}
	proxy := httputil.NewSingleHostReverseProxy(remote)
	http.HandleFunc("/", handler(proxy))
	slog.Info("Proxy is listening", "local", "http://"+from, "remote", "http://"+remote.Host)
	err := http.ListenAndServe(from, nil)
	if err != nil {
		panic(err)
	}
}
