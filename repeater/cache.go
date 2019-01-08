package repeater

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/dearcode/crab/cache"
	"github.com/dearcode/crab/log"
	"github.com/dearcode/crab/util/aes"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/meta"
	"github.com/dearcode/doodle/util"
)

type dbCache struct {
	cache          *cache.Cache
	selService     *sql.Stmt
	selIface       *sql.Stmt
	selVar         *sql.Stmt
	selApp         *sql.Stmt
	selRelation    *sql.Stmt
	instStats      *sql.Stmt
	instErrorStats *sql.Stmt
	dbc            *sql.DB
	sync.RWMutex
}

func (dc *dbCache) closeAll() {
	if dc.dbc != nil {
		dc.dbc.Close()
		dc.dbc = nil
	}
	if dc.selService != nil {
		dc.selService.Close()
		dc.selService = nil
	}
	if dc.selIface != nil {
		dc.selIface.Close()
		dc.selIface = nil
	}
	if dc.selVar != nil {
		dc.selVar.Close()
		dc.selVar = nil
	}
	if dc.selApp != nil {
		dc.selApp.Close()
		dc.selApp = nil
	}

	if dc.selRelation != nil {
		dc.selRelation.Close()
		dc.selRelation = nil
	}

	if dc.instStats != nil {
		dc.instStats.Close()
		dc.instStats = nil
	}

	if dc.instErrorStats != nil {
		dc.instErrorStats.Close()
		dc.instErrorStats = nil
	}

}

func (dc *dbCache) conectDB() error {
	var err error
	defer func() {
		if err != nil {
			dc.closeAll()
		}
	}()

	if dc.dbc, err = mdb.GetConnection(); err != nil {
		return errors.Trace(err)
	}

	if dc.selService, err = dc.dbc.Prepare("select id, validate, version from service where path=?"); err != nil {
		return errors.Trace(err)
	}

	if dc.selIface, err = dc.dbc.Prepare("select id, method, backend, email from interface where service_id = ? and path=?"); err != nil {
		return errors.Trace(err)
	}

	if dc.selVar, err = dc.dbc.Prepare("select postion, name, type, required from variable where interface_id = ?"); err != nil {
		return errors.Trace(err)
	}

	if dc.selApp, err = dc.dbc.Prepare("select name, email from application where id = ? and token=?"); err != nil {
		return errors.Trace(err)
	}

	if dc.selRelation, err = dc.dbc.Prepare("select id from relation where application_id = ? and interface_id=?"); err != nil {
		return errors.Trace(err)
	}

	if dc.instStats, err = dc.dbc.Prepare("insert into stats (iface_id, app_id, cnt, err, cost, event_time) values (?,?,?,?,?,?) ON DUPLICATE KEY UPDATE cnt = cnt + ?, cost =  cost + ?"); err != nil {
		return errors.Trace(err)
	}

	if dc.instErrorStats, err = dc.dbc.Prepare("insert into stats_error (session, iface_id, app_id, info, ctime) values (?,?,?,?,?)"); err != nil {
		return errors.Trace(err)
	}

	return nil
}

var (
	errInvalidPath   = errors.New("invalid path")
	errInvalidToken  = errors.New("invalid token")
	errNotFoundToken = errors.New("token not found")
	errNotFound      = errors.New("not found")
	errForbidden     = errors.New("forbidden")
)

const (
	maxRetry = 5
)

func (dc *dbCache) getApp(token string) (*meta.Application, error) {
	buf, err := aes.Decrypt(token, util.AesKey)
	if err != nil {
		return nil, errors.Annotatef(errInvalidToken, err.Error())
	}

	id, n := binary.Varint([]byte(buf))
	if n < 1 {
		return nil, fmt.Errorf("invalid token %s", token)
	}

	return dc.getApplication(id, token)
}

func (dc *dbCache) dbQuery(call func() error) (err error) {
	dc.Lock()
	defer dc.Unlock()

	for i := 0; i < maxRetry; i++ {
		if dc.dbc != nil {
			if err = call(); err == nil {
				return
			}
		}

		if err = dc.conectDB(); err != nil {
			log.Errorf("conenct db error:%s", err.Error())
		}
	}

	return
}

func (dc *dbCache) queryDB(s *sql.Stmt, arg []interface{}, res []interface{}) error {
	dc.Lock()
	defer dc.Unlock()

	//读数据库
	var rows *sql.Rows
	var err error

	for i := 0; i < maxRetry; i++ {
		if dc.dbc != nil {
			if rows, err = s.Query(arg...); err != nil {
				log.Errorf("retry:%v Query error:%v", i, err.Error())
				continue
			}
			break
		}

		if err = dc.conectDB(); err != nil {
			log.Errorf("conenct db error:%s", err.Error())
		}
	}

	if err != nil {
		return errors.Trace(err)
	}
	defer rows.Close()

	if !rows.Next() {
		return errors.Annotatef(errNotFound, "sql:%#v, argv:%#v", s, arg)
	}

	if err = rows.Scan(res...); err != nil {
		return errors.Trace(err)
	}

	return nil
}

func (dc *dbCache) getInterface(key string) (*meta.Interface, error) {
	if v := dc.cache.Get(key); v != nil {
		return v.(*meta.Interface), nil
	}

	ps := strings.Split(key, "/")
	if len(ps) < 3 {
		return nil, errors.Trace(errInvalidPath)
	}

	p := meta.Service{}
	if err := dc.queryDB(dc.selService, []interface{}{ps[1]}, []interface{}{&p.ID, &p.Validate, &p.Version}); err != nil {
		return nil, errors.Trace(err)
	}

	path := ps[2]
	if p.Version == 1 {
		//Faas接口
		path = key[len(ps[1])+1:]
		if idx := strings.LastIndex(path, "?"); idx != -1 {
			path = path[:idx]
		}
		log.Debugf("faas path:%s, src:%s", path, key)
	}

	i := meta.Interface{}
	if err := dc.queryDB(dc.selIface, []interface{}{p.ID, path}, []interface{}{&i.ID, &i.Method, &i.Backend, &i.Email}); err != nil {
		return nil, errors.Trace(err)
	}

	i.Path = path
	i.Service = p

	dc.cache.Add(key, &i)

	return &i, nil
}

func (dc *dbCache) validateRelation(appID, ifaceID int64) error {
	key := fmt.Sprintf("\x03%d%d", appID, ifaceID)
	if v := dc.cache.Get(key); v != nil {
		return nil
	}
	var id int64
	if err := dc.queryDB(dc.selRelation, []interface{}{appID, ifaceID}, []interface{}{&id}); err != nil {
		return errors.Trace(err)
	}
	dc.cache.Add(key, id)
	return nil
}

func (dc *dbCache) getApplication(id int64, token string) (*meta.Application, error) {
	key := fmt.Sprintf("\x01%d", id)
	if v := dc.cache.Get(key); v != nil {
		return v.(*meta.Application), nil
	}
	a := meta.Application{ID: id}
	if err := dc.queryDB(dc.selApp, []interface{}{id, token}, []interface{}{&a.Name, &a.Email}); err != nil {
		return nil, errors.Trace(err)
	}

	dc.cache.Add(key, &a)
	return &a, nil
}

func (dc *dbCache) getVariable(id int64) ([]*meta.Variable, error) {
	key := fmt.Sprintf("\x02%d", id)

	if vs := dc.cache.Get(key); vs != nil {
		return vs.([]*meta.Variable), nil
	}

	var rows *sql.Rows
	var err error

	if err = dc.dbQuery(func() error {
		rows, err = dc.selVar.Query(id)
		return err
	}); err != nil {
		return nil, errors.Trace(err)
	}

	defer rows.Close()

	var vs []*meta.Variable

	for rows.Next() {
		var v meta.Variable
		if err = rows.Scan(&v.Postion, &v.Name, &v.Type, &v.Required); err != nil {
			return nil, errors.Trace(err)
		}
		vs = append(vs, &v)
	}

	dc.cache.Add(key, vs)

	return vs, nil
}

func (dc *dbCache) executeDB(s *sql.Stmt, arg []interface{}) (res sql.Result, err error) {
	dc.Lock()
	defer dc.Unlock()

	for i := 0; i < maxRetry; i++ {
		if dc.dbc != nil {
			if res, err = s.Exec(arg...); err != nil {
				log.Errorf("retry:%v exec error:%v", i, err.Error())
				continue
			}
			return
		}

		if err = dc.conectDB(); err != nil {
			log.Errorf("conenct db error:%s", err.Error())
		}
	}

	return
}

func (dc *dbCache) insertDB(s *sql.Stmt, arg []interface{}) (id int64, err error) {
	var res sql.Result

	if res, err = dc.executeDB(s, arg); err != nil {
		return
	}

	if id, err = res.LastInsertId(); err != nil {
		return
	}
	return
}

func (dc *dbCache) insertStats(iface, app int64, count, errs int, cost int64) error {
	id, err := dc.insertDB(dc.instStats, []interface{}{iface, app, count, errs, cost, time.Now().Format("2006-01-02 15:04"), count, cost})
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("insert stats:%v", id)
	return nil
}

func (dc *dbCache) insertErrorStats(session string, iface, app int64, info string, tm time.Time) error {
	id, err := dc.insertDB(dc.instErrorStats, []interface{}{session, iface, app, info})
	if err != nil {
		return errors.Trace(err)
	}
	log.Debugf("insert error stats:%v", id)
	return nil
}
