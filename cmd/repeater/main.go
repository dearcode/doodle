package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dearcode/crab/log"

	"github.com/dearcode/doodle/repeater"
	"github.com/dearcode/doodle/repeater/config"
	"github.com/dearcode/doodle/util"
)

var (
	addr    = flag.String("h", ":8000", "api listen address")
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
		log.SetOutputFile("./logs/repeater.log")
		log.SetColor(false)
		log.SetRolling(true)
	}

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		panic(err.Error())
	}

	if err = repeater.Init(); err != nil {
		panic(err.Error())
	}

	as := http.Server{Handler: repeater.Server}

	go func() {
		if err = as.Serve(ln); err != nil {
			log.Error(err)
		}
	}()

	log.Infof("listen addr:%v", ln.Addr().String())

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGUSR1)

	s := <-shutdown
	log.Warningf("recv signal %v, close.", s)

	as.Shutdown(context.Background())
	repeater.Stop()

	time.Sleep(time.Duration(config.Repeater.Cache.Timeout) * time.Second)
	log.Warningf("server exit")
}
