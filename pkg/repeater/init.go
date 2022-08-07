package repeater

import (
	"dearcode.net/crab/cache"
	"dearcode.net/crab/orm"
	"github.com/juju/errors"

	"dearcode.net/doodle/pkg/repeater/config"
)

var (
	//Server 对外入口
	Server *repeater
	mdb    *orm.DB
	dc     *dbCache
	bs     *backendService
	stats  *statsCache
)

// repeater 网关验证模块
type repeater struct {
}

// Init 初始化HTTP接口.
func Init() error {
	if err := config.Load(); err != nil {
		return errors.Trace(err)
	}

	stats = newStatsCache()
	go stats.run()

	mdb = &config.Repeater.DB

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

// Stop 结束后端监控.
func Stop() {
	bs.stop()
}
