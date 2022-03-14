package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	port   = 4002
	key    = "Tenant-Id"
	direct = "Proxy-Direct"
)

var (
	log = ctrl.Log.WithName("Proxy")
)

type Proxy interface {
	Run(ctx context.Context)
}

type proxy struct {
	server *http.Server

	transport http.RoundTripper
}

func NewProxy(transport http.RoundTripper) (Proxy, error) {
	mux := http.NewServeMux()

	p := &proxy{
		transport: transport,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
	err := p.setHandle(mux)

	return p, err
}

type handler func(w http.ResponseWriter, r *http.Request)

func (p *proxy) setHandle(mux *http.ServeMux) error {
	mux.HandleFunc("/", do(p))
	return nil
}

func do(p *proxy) handler {
	return func(w http.ResponseWriter, r *http.Request) {
		value := r.Header.Get(direct)
		if value == "" {
			value = r.Header.Get(key)
			if value == "" {
				log.Info("unkonwn proxy", "uri", r.RequestURI, "method", r.Method)
				w.WriteHeader(http.StatusBadGateway)
				return
			}
		}

		uri := "http://" + value + "-" + r.Host
		log.Info("proxy", "uri", uri)

		url, err := url.ParseRequestURI(uri)
		if err != nil {
			log.Error(err, "parse request uri", "host", r.Host, "value", value)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Transport = p.transport
		proxy.ServeHTTP(w, r)
	}
}

func (p *proxy) Run(ctx context.Context) {
	doneCh := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			log.Info("proxy is shutting down")
			shutdownCtx, cancel := context.WithTimeout(
				context.Background(),
				time.Second*5,
			)
			defer cancel()
			p.server.Shutdown(shutdownCtx) // nolint: errcheck
		case <-doneCh:
		}
	}()

	log.Info("proxy is listening", "port", p.server.Addr)
	err := p.server.ListenAndServe()
	if err != http.ErrServerClosed {
		log.Error(err, "proxy error")
	}
	close(doneCh)
}
