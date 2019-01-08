package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"

	"github.com/dearcode/doodle/rbac"
	"github.com/dearcode/doodle/rbac/config"
	"github.com/dearcode/doodle/util"
)

var (
	addr    = flag.String("h", ":8100", "listen address")
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
		log.SetOutputFile("./logs/rbac.log")
		log.SetColor(false)
		log.SetRolling(true)
	}

	if err := rbac.ServerInit(); err != nil {
		panic(err)
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
	time.Sleep(time.Duration(config.RBAC.Server.Timeout) * time.Second)
	log.Warningf("server exit")
}
