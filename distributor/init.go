package distributor

import (
	"dearcode.net/crab/http/server"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/distributor/config"
)

var (
	mdb *orm.DB
	w   *watcher
)

// Init 初始化HTTP接口.
func Init(confPath string) error {
	var err error

	if err = config.Load(confPath); err != nil {
		return errors.Trace(err)
	}

	mdb = &config.Distributor.DB

	server.RegisterPath(&distributor{}, "/distributor/")

	if w, err = newWatcher(); err != nil {
		return errors.Trace(err)
	}

	go w.start()

	if err = w.load(); err != nil {
		return errors.Trace(err)
	}

	return nil
}

//Stop 关闭watcher.
func Stop() {
	w.stop()
}
