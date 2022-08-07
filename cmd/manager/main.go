package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"

	"dearcode.net/doodle/pkg/manager"
	"dearcode.net/doodle/pkg/manager/config"
	"dearcode.net/doodle/pkg/util"
)

var (
	addr    = flag.String("h", ":8200", "listen address")
	debug   = flag.Bool("debug", false, "debug write log to console.")
	version = flag.Bool("v", false, "show version info")
)

func main() {
	flag.Parse()

	if *version {
		util.PrintVersion()
		return
	}

	if !*debug {
		log.SetOutputFile("./logs/manager.log")
		log.SetColor(false)
		log.SetRolling(true)
		log.SetLevel(log.LogDebug)
	}

	if err := manager.ServerInit(); err != nil {
		panic(err.Error())
	}

	ln, err := server.Start(*addr)
	if err != nil {
		panic(err)
	}

	log.Infof("listener %s", ln.Addr())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGUSR1)

	s := <-shutdown
	log.Warningf("recv signal %v, close.", s)
	ln.Close()
	time.Sleep(time.Duration(config.Manager.Cache.Timeout) * time.Second)
	log.Warningf("server exit")
}
