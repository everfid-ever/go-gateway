package main

import (
	"flag"
	"go-gateway/common/lib"
	"go-gateway/router"
	"os"
	"os/signal"
	"syscall"
)

var endpoint = flag.String("endpoint", "", "dashboard or server")

func main() {
	flag.Parse()
	if *endpoint == "" {
		flag.Usage()
		os.Exit(1)
	}

	if *endpoint == "dashboard" {
		lib.InitModule(*endpoint)
		defer lib.Destroy()
		router.HttpServerRun()

		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		router.HttpServerStop()
	}
}
