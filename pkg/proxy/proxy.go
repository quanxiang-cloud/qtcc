package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/quanxiang-cloud/qtcc/pkg/apis/qtcc/v1alpha1"
	scheme "github.com/quanxiang-cloud/qtcc/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	client scheme.Interface

	transport http.RoundTripper

	namespace string
}

func NewProxy(transport http.RoundTripper, client scheme.Interface, opts ...Option) (Proxy, error) {
	mux := http.NewServeMux()

	p := &proxy{
		client:    client,
		transport: transport,
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}

	for _, opt := range opts {
		opt(p)
	}

	err := p.setHandle(mux)

	return p, err
}

type Option func(p Proxy)

func WithNamespace(namespace string) Option {
	return func(p Proxy) {
		proxy, ok := p.(*proxy)
		if ok {
			proxy.namespace = namespace
		}
	}
}

type handler func(w http.ResponseWriter, r *http.Request)

func (p *proxy) setHandle(mux *http.ServeMux) error {
	mux.HandleFunc("/", do(p))
	mux.HandleFunc("/whereami", p.whereami)
	return nil
}

func (p *proxy) whereami(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	ctx := context.Background()
	kind := r.URL.Query().Get("kind")
	if kind == "" {
		log.Info("kind is must.")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	tenantID := r.Header.Get(key)
	directKey := r.Header.Get(direct)
	if tenantID == "" {
		if kind == "" {
			log.Info("tenant id is must.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

	}
	log.Info("get config info", "tenantID", tenantID, "kind", kind)

	value := fmt.Sprintf("%s-%s", tenantID, kind)
	if directKey != "" {
		value = kind
	}

	config, err := p.client.QtccV1alpha1().Configs(p.namespace).Get(ctx, value, v1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "get config", "tenantID", tenantID, "kind", kind)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		err = nil
		config = nil
	}

	resp := &struct {
		Host string `json:"host,omitempty"`
		v1alpha1.RollingWay
	}{}

	if config != nil {
		resp.Host = fmt.Sprintf("http://%s.%s.svc.cluster.local", value, p.namespace)
		resp.RollingWay = config.Spec.RollingWay
	}

	body, err := json.Marshal(resp)
	if err != nil {
		log.Error(err, "json mashal", "resp", resp, "tenantID", tenantID, "kind", kind)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	_, err = w.Write(body)
	if err != nil {
		log.Error(err, "write response", "tenantID", tenantID, "kind", kind)
	}
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

		uri := fmt.Sprintf("http://%s-%s.%s.svc.cluster.local", value, r.Host, p.namespace)
		log.Info("proxy", "uri", uri)

		url, err := url.ParseRequestURI(uri)
		if err != nil {
			log.Error(err, "parse request uri", "host", r.Host, "value", value)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(url)
		proxy.Transport = p.transport
		r.Host = url.Host
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
