package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"server-saver/internal/director"
	"server-saver/internal/launcher"
	"syscall"

	"server-saver/internal/proxy"

	"github.com/alecthomas/kong"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

var CLI struct {
	Config  string `kong:"arg,help='path to configuration file to load',type='path'"`
	Verbose bool   `kong:"help='enable verbose logs'"`
}

var config struct {
	BindAddress           string `json:"bind_address"`
	LauncherConfiguration struct {
		SelectedLauncher string                     `json:"selected_launcher"`
		SimpleProxy      launcher.SimpleProxyConfig `json:"simple_proxy"`
		Executable       launcher.ExecutableConfig  `json:"executable"`
	} `json:"launcher_configuration"`
}

func main() {
	kong.Parse(&CLI)

	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Sprintf("failed to initialise logger: %v", err))
	}
	logger := zapr.NewLogger(zapLogger).WithName("server-saver")

	ctx, cancel := context.WithCancel(context.Background())
	group, ctx := errgroup.WithContext(ctx)

	cfgFile, err := os.Open(CLI.Config)
	if err != nil {
		fmt.Printf("unable to open configuration: %v\n", err)
		os.Exit(1)
	}
	dec := json.NewDecoder(cfgFile)
	if err := dec.Decode(&config); err != nil {
		fmt.Printf("unable to parse configuration file: %v\n", err)
		os.Exit(1)
	}

	group.Go(func() error {
		l := logger.WithName("exit-handler")

		s := make(chan os.Signal)
		signal.Notify(s, syscall.SIGTERM, syscall.SIGINT)

		l.Info("waiting for exit signal")
		<-s
		l.Info("exit signal received")
		cancel()

		return nil
	})

	var l launcher.Launcher

	if config.LauncherConfiguration.SelectedLauncher == "simple-proxy" {
		l = launcher.NewSimpleProxy(config.LauncherConfiguration.SimpleProxy)
	} else if config.LauncherConfiguration.SelectedLauncher == "executable" {
		l = launcher.NewExecutableLauncher(
			ctx,
			logger.WithName("executable-launcher"),
			config.LauncherConfiguration.Executable,
		)
	}

	d := director.New(ctx, logger.WithName("director"), l)

	p := proxy.Server{
		OnConnect: func(info proxy.ConnInfo) {
			d.RegisterConnection(info.Uid())
		},
		OnDisconnect: func(info proxy.ConnInfo) {
			d.UnregisterConnection(info.Uid())
		},
		Target: l,
	}
	group.Go(func() error {
		return p.Proxy(ctx, logger.WithName("proxy-server"), config.BindAddress)
	})

	if err := group.Wait(); err != nil {
		fmt.Printf("error: %v\n", err)
	}
}
