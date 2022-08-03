package rbac

import (
	"encoding/binary"
	"html/template"
	"net/http"
	"runtime"
	"strings"

	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"
	"dearcode.net/crab/util/aes"
	"github.com/juju/errors"

	"dearcode.net/doodle/rbac/config"
)

var (
	errInvalidToken = errors.New("invalid token")
	mdb             *orm.DB
)

// ServerInit 初始化HTTP接口.
func ServerInit() error {
	if err := config.Load("rbac.ini"); err != nil {
		return err
	}

	mdb = &config.RBAC.DB

	server.RegisterPath(&rbacUser{}, "/rbac/user/")
	server.RegisterPath(&rbacUserInfo{}, "/rbac/user/info/")
	server.RegisterPath(&userRole{}, "/rbac/user/role/")
	server.RegisterPath(&rbacRoleUser{}, "/rbac/role/user/")
	server.RegisterPath(&rbacRoleResource{}, "/rbac/role/resource/")
	server.RegisterPath(&resourceRolesUnrelated{}, "/rbac/resource/role/unrelated/")
	server.RegisterPath(&rbacRole{}, "/rbac/role/")
	server.RegisterPath(&rbacAPP{}, "/rbac/app/")
	server.RegisterPath(&rbacResource{}, "/rbac/resource/")
	server.RegisterPath(&userResource{}, "/rbac/user/resource/")

	server.RegisterPrefix(&static{}, "/static/")

	server.RegisterPath(&authorize{}, "/authorize/")
	server.RegisterPath(&account{}, "/account/")

	return nil
}

func parseToken(r *http.Request) (int64, error) {
	token := r.Header.Get("token")
	buf, err := aes.Decrypt(token, config.RBAC.Server.Key)
	if err != nil {
		return 0, errors.Annotatef(errInvalidToken, "token:%v, error:%v", token, err.Error())
	}

	id, n := binary.Varint([]byte(buf))
	if n < 1 {
		return 0, errors.Errorf("invalid token %s", token)
	}

	return id, nil
}

type static struct {
}

//GET 下载静态文件
func (s *static) GET(w http.ResponseWriter, r *http.Request) {
	//	w.Header().Add("Cache-control", "no-store")
	log.Debugf("file:%v", config.RBAC.Server.WebPath+r.URL.RequestURI())
	http.ServeFile(w, r, config.RBAC.Server.WebPath+r.URL.RequestURI())
}

func execute(w http.ResponseWriter, data interface{}) {
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc).Name()
	fna := strings.Split(fn, ".")
	name := fna[len(fna)-2]
	name = strings.Replace(name, "*", "", -1)
	name = strings.Replace(name, "(", "", -1)
	name = strings.Replace(name, ")", "", -1)

	t, err := template.ParseFiles(config.RBAC.Server.WebPath + "/" + name + ".html")
	if err != nil {
		server.Abort(w, err.Error())
		return
	}

	if err := t.Execute(w, data); err != nil {
		server.Abort(w, err.Error())
	}
}
