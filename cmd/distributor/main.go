package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"

	"dearcode.net/doodle/distributor"
	"dearcode.net/doodle/util"
)

var (
	addr       = flag.String("h", ":8300", "api listen address")
	debug      = flag.Bool("debug", false, "debug write log to console.")
	version    = flag.Bool("v", false, "show version info")
	configPath = flag.String("c", "./config/distributor.ini", "config file")

	maxWaitTime = time.Minute
)

func main() {
	flag.Parse()

	if *version {
		util.PrintVersion()
		return
	}

	if !*debug {
		log.SetOutputFile("./logs/distributor.log")
		log.SetColor(false)
		log.SetRolling(true)
	}

	if err := distributor.Init(*configPath); err != nil {
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

	distributor.Stop()

	ln.Close()
	time.Sleep(maxWaitTime)
	log.Warningf("server exit")
}
