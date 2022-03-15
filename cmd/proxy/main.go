package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/quanxiang-cloud/qtcc/pkg/client/clientset/versioned"
	"github.com/quanxiang-cloud/qtcc/pkg/proxy"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	log = ctrl.Log.WithName("Injector")
)

func main() {
	var maxIdle int
	var maxIdlePerHost int
	var namespace string
	flag.IntVar(&maxIdle, "maxIdle", 100, "max idle.")
	flag.IntVar(&maxIdlePerHost, "maxIdlePerHost", 100, "max idle per host.")
	flag.StringVar(&namespace, "namespace", "lowcode", "proxy namespace")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info(`Received signal; beginning shutdown`)
		cancel()
	}()

	kubeConfig := ctrl.GetConfigOrDie()

	transport := proxy.NewTransport(maxIdle, maxIdlePerHost)
	proxy, err := proxy.NewProxy(
		transport,
		versioned.NewForConfigOrDie(kubeConfig),
		proxy.WithNamespace(namespace),
	)
	if err != nil {
		log.Error(err, "new proxy")
		return
	}

	log.Info("starting proxy")
	proxy.Run(ctx)

}
