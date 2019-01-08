package distributor

import (
	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/orm"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/distributor/config"
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

	mdb = orm.NewDB(config.Distributor.DB.IP, config.Distributor.DB.Port, config.Distributor.DB.Name, config.Distributor.DB.User, config.Distributor.DB.Passwd, config.Distributor.DB.Charset, 10)

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
