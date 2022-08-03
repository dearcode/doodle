package repeater

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"dearcode.net/crab/log"
	"github.com/juju/errors"
	"go.etcd.io/etcd/client/v3"

	"dearcode.net/doodle/meta"
	"dearcode.net/doodle/repeater/config"
	"dearcode.net/doodle/util/etcd"
)

const (
	apigatePrefix = "/api"
)

type backendService struct {
	etcd *etcd.Client
	apps map[string][]meta.MicroAPP
	mu   sync.RWMutex
}

func newBackendService() (*backendService, error) {
	c, err := etcd.New(config.Repeater.ETCD.Hosts)
	if err != nil {
		return nil, errors.Annotatef(err, config.Repeater.ETCD.Hosts)
	}

	return &backendService{etcd: c, apps: make(map[string][]meta.MicroAPP)}, nil
}

func (bs *backendService) start() {
	ec := make(chan clientv3.Event)

	for {
		go bs.etcd.WatchPrefix(apigatePrefix, ec)
		for e := range ec {
			ss := strings.Split(string(e.Kv.Key), "/")
			if len(ss) < 4 {
				log.Errorf("invalid key:%s, event:%v", e.Kv.Key, e.Type)
				continue
			}

			//type只有DELETE和PUT.
			name := strings.Join(ss[2:len(ss)-2], "/")
			if e.Type == clientv3.EventTypeDelete {
				port, _ := strconv.Atoi(ss[len(ss)-1])
				bs.unregister(name, ss[len(ss)-2], port)
				continue
			}

			app := meta.MicroAPP{}
			json.Unmarshal(e.Kv.Value, &app)
			bs.register(name, app)
		}
	}
}

func (bs *backendService) load() error {
	bss, err := bs.etcd.List(apigatePrefix)
	if err != nil {
		log.Errorf("list %s error:%v", apigatePrefix, err)
		return errors.Annotatef(err, apigatePrefix)
	}

	for k, v := range bss {
		// k = /api/dbs/dbfree/handler/Fore/192.168.180.102/21638
		ss := strings.Split(k, "/")
		if len(ss) < 4 {
			log.Errorf("invalid key:%s", k)
			continue
		}

		//type只有DELETE和PUT.
		name := strings.Join(ss[2:len(ss)-2], "/")
		app := meta.MicroAPP{}
		json.Unmarshal([]byte(v), &app)
		bs.register(name, app)
	}

	return nil
}

//unregister 如果etcd中事务是删除，这里就去管理处删除.
func (bs *backendService) unregister(name, host string, port int) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	apps, ok := bs.apps[name]
	if !ok {
		log.Debugf("cache app:%s not found", name)
		return
	}

	for i, app := range apps {
		if app.Host == host && app.Port == port {
			log.Infof("remove app:%s, addr:%v:%v", name, host, port)
			//只有他自己，直接删除了.
			if len(apps) == 1 {
				delete(bs.apps, name)
				return
			}
			ns := []meta.MicroAPP{}
			ns = append(ns, apps[:i]...)
			ns = append(ns, apps[i+1:]...)
			bs.apps[name] = ns
			return
		}
	}
}

//register 到管理处添加接口, 肯定是多个repeater同时上报的，所以添加操作要指定版本信息.
func (bs *backendService) register(name string, app meta.MicroAPP) {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	apps, ok := bs.apps[name]
	if !ok {
		bs.apps[name] = []meta.MicroAPP{app}
		log.Debugf("new name:%s, app:%+v", name, app)
		return
	}

	for _, o := range apps {
		if o.Host == app.Host && o.Port == app.Port {
			log.Errorf("invalid app:%v, apps:%#v", app, apps)
			return
		}
	}

	bs.apps[name] = append(apps, app)

	log.Debugf("new name:%s, add app:%+v", name, app)
}

//getMicroAPPs 根据接口名获取后端应用列表.
func (bs *backendService) getMicroAPPs(name string) ([]meta.MicroAPP, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	log.Debugf("find name:%v", name)
	for k, v := range bs.apps {
		log.Debugf("k:%v, v:%v", k, v)
	}

	apps, ok := bs.apps[name]
	if !ok {
		return nil, errors.Annotatef(errNotFound, name)
	}

	return apps, nil
}

func (bs *backendService) stop() {
	bs.etcd.Close()
}
