package distributor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"dearcode.net/crab/http/client"
	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/util/aes"
	"github.com/juju/errors"
	"go.etcd.io/etcd/client/v3"

	"dearcode.net/doodle/distributor/config"
	"dearcode.net/doodle/meta"
	"dearcode.net/doodle/meta/document"
	"dearcode.net/doodle/util/etcd"
)

const (
	apigatePrefix = "/api"
)

type microService struct {
	version int64
	nodes   map[string]meta.MicroAPP
}

type watcher struct {
	etcd     *etcd.Client
	services map[string]microService
	mu       sync.RWMutex
}

func newWatcher() (*watcher, error) {
	c, err := etcd.New(config.Distributor.ETCD.Hosts)
	if err != nil {
		return nil, errors.Annotatef(err, config.Distributor.ETCD.Hosts)
	}

	return &watcher{etcd: c, services: make(map[string]microService)}, nil
}

func (w *watcher) start() {
	ec := make(chan clientv3.Event)

	for {
		go w.etcd.WatchPrefix(apigatePrefix, ec)
		for e := range ec {
			// /api/dbs/dbfree/handler/Fore/192.168.180.102/21638
			ss := strings.Split(string(e.Kv.Key), "/")
			if len(ss) < 4 {
				log.Errorf("invalid key:%s, event:%v", e.Kv.Key, e.Type)
				continue
			}

			name := strings.Join(ss[2:len(ss)-2], "/")
			host := ss[len(ss)-2]

			if e.Type == clientv3.EventTypeDelete {
				w.offline(name, host)
				continue
			}

			app := meta.MicroAPP{}
			json.Unmarshal(e.Kv.Value, &app)

			w.online(name, app)
		}
	}
}

func (w *watcher) load() error {
	bss, err := w.etcd.List(apigatePrefix)
	if err != nil {
		log.Errorf("list %s error:%v", apigatePrefix, err)
		return errors.Annotatef(err, apigatePrefix)
	}

	for k, v := range bss {
		ss := strings.Split(k, "/")
		if len(ss) < 4 {
			log.Debugf("invalid key:%s", k)
			continue
		}

		name := strings.Join(ss[2:len(ss)-2], "/")
		app := meta.MicroAPP{}
		json.Unmarshal([]byte(v), &app)
		w.online(name, app)
	}

	return nil
}

type managerClient struct {
}

func (mc *managerClient) interfaceRegister(serviceID, version int64, name, method, path, backend string, m document.Method) error {
	url := fmt.Sprintf("%sinterface/register/", config.Distributor.Manager.URL)
	req := struct {
		Name      string
		ServiceID int64
		Version   int64
		Path      string
		Method    server.Method
		Backend   string
		Comment   string
		Attr      document.Method
	}{
		Name:      name,
		ServiceID: serviceID,
		Version:   version,
		Path:      path,
		Backend:   backend,
		Method:    server.NewMethod(method),
		Comment:   m.Comment,
		Attr:      m,
	}

	resp := struct {
		Status  int
		Data    int
		Message string
	}{}

	if err := client.New().Timeout(config.Distributor.Server.Timeout).PostJSON(url, nil, req, &resp); err != nil {
		return errors.Annotatef(err, url)
	}

	if resp.Status != 0 {
		return errors.New(resp.Message)
	}

	log.Debugf("register %+v success, id:%v", req, resp.Data)

	return nil
}

const (
	httpConnectTimeout = 60
)

func (w *watcher) parseDocument(backend string, app meta.MicroAPP) error {
	url := fmt.Sprintf("http://%s:%d/document/", app.Host, app.Port)
	buf, err := client.New().Timeout(httpConnectTimeout).Get(url, nil, nil)
	if err != nil {
		return errors.Trace(err)
	}

	var doc map[string]document.Module
	log.Debugf("source:%s", buf)

	if err = json.Unmarshal(buf, &doc); err != nil {
		log.Errorf("Unmarshal doc:%s error:%v", buf, err)
		return errors.Annotatef(err, "%s", buf)
	}

	log.Debugf("doc:%+v", doc)

	serviceID, err := parseServiceID(app.ServiceKey)
	if err != nil {
		log.Errorf("parseServiceID:%s error:%v", app.ServiceKey, err)
		return errors.Annotatef(err, app.ServiceKey)
	}

	version, err := strconv.ParseInt(app.GitTime, 10, 64)
	if err != nil {
		log.Errorf("parse GitTime:%v error:%v", app.GitTime, err)
		return errors.Annotatef(err, app.GitTime)
	}

	mc := managerClient{}
	for ok, ov := range doc {
		for mk, mv := range ov.Methods {
			mc.interfaceRegister(serviceID, version, ok+"_"+mk, mk, ov.URL, backend, mv)
		}
	}

	return nil
}

//online 到管理处添加接口, 肯定是多个Distributor同时上报的，所以添加操作要指定版本信息.
func (w *watcher) online(backend string, app meta.MicroAPP) {
	w.mu.Lock()
	defer w.mu.Unlock()

	o, ok := w.services[backend]
	if ok {
		o.nodes[app.Host] = app
		log.Debugf("backend:%v, add host:%v", backend, app.Host)

		//如果是版本未变或旧版本就添加到节点列表中就返回.
		if app.Version() <= o.version {
			log.Debugf("app:%+v exist", app)
			return
		}

	} else {
		//如果后端接口不存在, 添加、注册
		o = microService{nodes: make(map[string]meta.MicroAPP)}
		o.nodes[app.Host] = app
		log.Debugf("backend:%v, add host:%v, new app", backend, app.Host)
	}

	o.version = app.Version()
	w.services[backend] = o
	w.parseDocument(backend, app)
	log.Debugf("new backend:%s, app:%+v", backend, app)
}

func (w *watcher) stop() {
	w.etcd.Close()
}

func parseServiceID(key string) (int64, error) {
	buf, err := aes.Decrypt(key, config.Distributor.Server.SecretKey)
	if err != nil {
		return 0, errors.Trace(err)
	}

	var id int64
	_, err = fmt.Sscanf(string(buf), "%x.", &id)
	if err != nil {
		return 0, errors.Trace(err)
	}

	return id, nil
}

func (w *watcher) get(name, host string) meta.MicroAPP {
	w.mu.RLock()
	defer w.mu.RUnlock()

	s, ok := w.services[name]
	if !ok {
		log.Debugf("not found name:%v", name)
		return meta.MicroAPP{}
	}

	a, ok := s.nodes[host]
	if !ok {
		log.Debugf("not found name:%v, host:%v", name, host)
		return meta.MicroAPP{}
	}
	return a
}

func (w *watcher) offline(name, host string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	s, ok := w.services[name]
	if !ok {
		log.Infof("name:%v not found", name)
		return
	}

	log.Infof("name:%v, host:%v", name, host)

	delete(s.nodes, host)
}
