package service

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/dearcode/crab/http/server"
	"github.com/dearcode/crab/log"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/service/debug"
)

//RequestHeader 默认请求头.
type RequestHeader struct {
	Session string
	Request http.Request
}

//String session id.
func (h RequestHeader) String() string {
	return h.Session
}

//ResponseHeader 默认返回头.
type ResponseHeader struct {
	Status  int
	Message string `json:",omitempty"`
}

//Service 一个服务对象.
type Service struct {
	doc     document
	docView docView
	router  router
}

var (
	host        = flag.String("h", ":8080", "listen address.")
	version     = flag.Bool("V", false, "version info.")
	logLevel    = flag.String("logLevel", "debug", "log level: fatal, error, warning, debug, info.")
	logFile     = flag.String("logFile", "", "log file name.")
	etcdAddrs   = flag.String("etcd", "", "etcd Endpoints, like 192.168.180.104:12379,192.168.180.104:22379,192.168.180.104:32379.")
	maxWaitTime = time.Hour * 24
)

//New 返回service对象.
func New() *Service {
	return &Service{
		doc:    newDocument(),
		router: newRouter(),
	}
}

//Init 解析flag参数, 初始化基本信息.
func (s *Service) Init() {
	flag.Parse()

	if *version {
		debug.Print()
		os.Exit(0)
	}

	if *logFile != "" {
		log.SetColor(false)
		log.SetRolling(true)
		log.SetOutputFile(*logFile)
	}

	log.SetLevelByString(*logLevel)

	server.RegisterPrefix(&debug.Debug{}, "/debug/pprof/")
	server.RegisterPrefix(&debug.Version{}, "/debug/version/")
	server.RegisterPrefix(&s.doc, "/document/")

}

//Register 注册接口.
func (s *Service) Register(obj interface{}) error {
	t := reflect.TypeOf(obj)
	name := t.Name()
	pkg := t.PkgPath()

	//不能脱壳，脱壳后取不到method.
	if t.Kind() == reflect.Ptr {
		name = t.Elem().Name()
		pkg = t.Elem().PkgPath()
	}

	pkg = path.Base(pkg)
	url := fmt.Sprintf("/%s/%s/", pkg, name)
	log.Debugf("url:%v", url)

	for _, k := range []string{"Get", "Post", "Put", "Delete"} {
		if m, ok := t.MethodByName(k); ok {
			if m.Type.NumIn() == 3 {
				if err := server.RegisterHandler(s.handler, strings.ToUpper(k), url); err != nil {
					log.Errorf("RegisterPrefix %v error:%v", url, err)
					return err
				}
				s.router.add(strings.ToUpper(k), url, m)
				s.doc.add(name, url, m)
			}
		}
	}

	return nil
}

//Start 开启服务.
func (s *Service) Start() {
	s.docView = s.doc.view()
	server.RegisterPath(&s.docView, "/doc/")

	//第一步，启动服务
	ln, err := server.Start(*host)
	if err != nil {
		log.Errorf("%v", errors.ErrorStack(err))
		panic(err)
	}

	//第二步，注册到接口平台API接口队列中.
	keepalive, err := newKeepalive(*etcdAddrs, ln.Addr().String())
	if err != nil {
		log.Errorf("apiRegister error:%v", errors.ErrorStack(err))
		panic(err)
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGUSR1)

	log.Infof("service:%v", ln.Addr())

	sig := <-shutdown
	keepalive.stop()
	log.Warningf("%v recv signal %v, close:%v", os.Getpid(), sig, ln.Close())

	log.Warningf("%v wait timeout:%v.", os.Getpid(), maxWaitTime)
	<-time.After(maxWaitTime)
	log.Warningf("%v exit", os.Getpid())
}

//SetError 返回错误信息.
func (h *ResponseHeader) SetError(status int, format string, argv ...interface{}) {
	h.Status = status
	h.Message = fmt.Sprintf(format, argv...)
}

//InvalidRequest 返回因为请求参数错误导致的异常信息(400).
func (h *ResponseHeader) InvalidRequest(format string, argv ...interface{}) {
	h.Status = http.StatusBadRequest
	h.Message = fmt.Sprintf(format, argv...)
}

//InternalError 返回因为内部错误导致的异常信息(500).
func (h *ResponseHeader) InternalError(format string, argv ...interface{}) {
	h.Status = http.StatusInternalServerError
	h.Message = fmt.Sprintf(format, argv...)
}
