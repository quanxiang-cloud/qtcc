package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

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
	flag.IntVar(&maxIdle, "maxIdle", 100, "max idle.")
	flag.IntVar(&maxIdlePerHost, "maxIdlePerHost", 100, "max idle per host.")
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

	transport := proxy.NewTransport(maxIdle, maxIdlePerHost)

	proxy, err := proxy.NewProxy(transport)
	if err != nil {
		log.Error(err, "new proxy")
		return
	}

	proxy.Run(ctx)

	log.Info("starting proxy")
}
