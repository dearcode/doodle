package repeater

import (
	"github.com/dearcode/crab/cache"
	"github.com/dearcode/crab/orm"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/repeater/config"
)

var (
	//Server 对外入口
	Server *repeater
	mdb    *orm.DB
	dc     *dbCache
	bs     *backendService
	stats  *statsCache
)

//repeater 网关验证模块
type repeater struct {
}

// Init 初始化HTTP接口.
func Init() error {
	if err := config.Load(); err != nil {
		return errors.Trace(err)
	}

	stats = newStatsCache()
	go stats.run()

	mdb = orm.NewDB(config.Repeater.DB.IP, config.Repeater.DB.Port, config.Repeater.DB.Name, config.Repeater.DB.User, config.Repeater.DB.Passwd, config.Repeater.DB.Charset, 10)

	dc = &dbCache{cache: cache.NewCache(int64(config.Repeater.Cache.Timeout))}
	if err := dc.conectDB(); err != nil {
		return errors.Trace(err)
	}

	Server = &repeater{}

	nbs, err := newBackendService()
	if err != nil {
		return errors.Trace(err)
	}

	go nbs.start()

	if err := nbs.load(); err != nil {
		return errors.Trace(err)
	}

	bs = nbs

	return nil
}

//Stop 结束后端监控.
func Stop() {
	bs.stop()
}
