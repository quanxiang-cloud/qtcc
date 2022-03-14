package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/quanxiang-cloud/qtcc/pkg/client/clientset/versioned"
	"github.com/quanxiang-cloud/qtcc/pkg/injector"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	log = ctrl.Log.WithName("Injector")
)

func main() {
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	conf, err := injector.GetConfig()
	if err != nil {
		log.Error(err, "get config")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info(`Received signal; beginning shutdown`)
		cancel()
	}()

	kubeConfig := ctrl.GetConfigOrDie()

	log.Info("starting injector")
	injector.NewInjector(
		conf,
		versioned.NewForConfigOrDie(kubeConfig),
		kubernetes.NewForConfigOrDie(kubeConfig),
	).Run(ctx)
}
