package repeater

import (
	"sync"
	"time"

	"github.com/dearcode/crab/log"

	"github.com/dearcode/doodle/repeater/config"
)

type errorEntry struct {
	Session string
	App     int64
	Iface   int64
	Info    string
	Time    time.Time
}

type entry struct {
	App   int64
	Iface int64
	Count int
	Err   int
	Time  int64
}

type ifaceEntry struct {
	apps map[int64]*entry
}

type statsCache struct {
	access map[int64]*ifaceEntry
	errors []*errorEntry
	sync.Mutex
}

func newStatsCache() *statsCache {
	return &statsCache{access: make(map[int64]*ifaceEntry)}
}

func (s *statsCache) success(app, iface, tm int64) {
	s.add("", app, iface, tm, true, "")
}

func (s *statsCache) failed(id string, app, iface int64, msg string) {
	s.add(id, app, iface, 0, false, msg)
}

//add 添加记录, 合并同一app调用同一接口的统计
func (s *statsCache) add(id string, app, iface, tm int64, success bool, msg string) {
	s.Lock()
	defer s.Unlock()

	ie, ok := s.access[iface]
	if !ok {
		ie = &ifaceEntry{apps: make(map[int64]*entry)}
		s.access[iface] = ie
	}

	e, ok := ie.apps[app]
	if !ok {
		e = &entry{
			App:   app,
			Iface: iface,
		}
		ie.apps[app] = e
	}

	e.Count++
	if !success {
		s.errors = append(s.errors, &errorEntry{id, app, iface, msg, time.Now().Add(time.Hour * 8)})
		e.Err++
	}
	e.Time += tm
	log.Debugf("new log:%+v", *e)
}

//entrys 读取统计信息, 并清理
func (s *statsCache) entrys() []entry {
	s.Lock()
	defer s.Unlock()

	var es []entry

	for ii, ie := range s.access {
		for ai, e := range ie.apps {
			es = append(es, *e)
			delete(ie.apps, ai)
		}
		delete(s.access, ii)
	}

	return es
}

//errorEntrys 异常日志, 并清理
func (s *statsCache) errorEntrys() []*errorEntry {
	s.Lock()
	defer s.Unlock()

	errs := s.errors
	s.errors = []*errorEntry{}
	return errs
}

func (s *statsCache) run() {
	t := time.NewTicker(time.Duration(config.Repeater.Cache.Timeout) * time.Second)
	for {
		<-t.C
		for _, e := range s.entrys() {
			if err := dc.insertStats(e.Iface, e.App, e.Count, e.Err, e.Time); err != nil {
				log.Errorf("insertStats %v error:%v", e, err.Error())
			}
		}

		for _, e := range s.errorEntrys() {
			if err := dc.insertErrorStats(e.Session, e.Iface, e.App, e.Info, e.Time); err != nil {
				log.Errorf("insertErrorStats %v error:%v", e, err.Error())
			}
		}
	}

}
