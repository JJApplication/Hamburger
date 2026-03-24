package main

import (
	"Hamburger/exp/trojan"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	configPath := ""
	flag.StringVar(&configPath, "c", "", "trojan config file path")
	flag.StringVar(&configPath, "config", "", "trojan config file path")
	flag.Parse()
	if configPath == "" {
		_, _ = os.Stderr.WriteString("missing required flag: -c or -config\n")
		os.Exit(1)
	}

	server, err := trojan.NewTrojanServerFromConfig(configPath)
	if err != nil {
		_, _ = os.Stderr.WriteString(fmt.Sprintf("init trojan server failed: %v\n", err))
		os.Exit(1)
	}
	if server == nil {
		_, _ = os.Stderr.WriteString("trojan server disabled\n")
		os.Exit(1)
	}

	startErr := make(chan error, 1)
	go func() {
		startErr <- server.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case err = <-startErr:
		if err != nil {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("trojan server exited: %v\n", err))
			os.Exit(1)
		}
	case <-sigCh:
		if stopErr := server.Stop(); stopErr != nil {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("stop trojan server failed: %v\n", stopErr))
			os.Exit(1)
		}
		if err = <-startErr; err != nil {
			_, _ = os.Stderr.WriteString(fmt.Sprintf("trojan server exited: %v\n", err))
			os.Exit(1)
		}
	}
}
