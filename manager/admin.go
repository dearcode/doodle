package manager

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/dearcode/crab/cache"
	"github.com/dearcode/crab/log"
	"github.com/dearcode/crab/orm"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/manager/config"
	"github.com/dearcode/doodle/rbac/meta"
)

type userDB struct {
	admins *cache.Cache
	res    *cache.Cache
	users  *cache.Cache
	roles  *cache.Cache
	sync.RWMutex
}

func newUserDB() *userDB {
	return &userDB{
		admins: cache.NewCache(int64(config.Manager.Cache.Timeout)),
		res:    cache.NewCache(int64(config.Manager.Cache.Timeout)),
		users:    cache.NewCache(int64(config.Manager.Cache.Timeout)),
		roles:  cache.NewCache(int64(config.Manager.Cache.Timeout)),
	}
}

//isAdmin 判断是不是管理员
func (u *userDB) isAdmin(email string) bool {
	u.RLock()
	if ok := u.admins.Get(email); ok != nil {
		u.RUnlock()
		log.Debugf("email:%v, cache:%v", email, ok.(bool))
		return ok.(bool)
	}
	u.RUnlock()

	u.Lock()
	defer u.Unlock()

	if ok := u.admins.Get(email); ok != nil {
		log.Debugf("retry email:%v, cache:%v", email, ok.(bool))
		return ok.(bool)
	}

	db, err := mdb.GetConnection()
	if err != nil {
		log.Errorf("get db connection error:%v", errors.ErrorStack(err))
		return false
	}
	defer db.Close()

	admin := struct {
		User  string `db:"user"`
		Email string `db:"email"`
	}{}

	if err = orm.NewStmt(db, "admin").Where("email='%s'", email).Query(&admin); err != nil {
		if orm.IsNotFound(err) {
			log.Debugf("%s not admin", email)
			u.admins.Add(email, false)
			return false
		}
		log.Errorf("orm query error:%v", errors.ErrorStack(err))
		return false
	}

	u.admins.Add(email, true)

	log.Debugf("%v admin:%v", email, admin)

	return true
}

//loadResource 查找用户权限
func (u *userDB) loadResource(email string) ([]int64, error) {
	u.RLock()
	if res := u.res.Get(email); res != nil {
		u.RUnlock()
		log.Debugf("user:%v, resource cache:%v", email, res.([]int64))
		return res.([]int64), nil
	}
	u.RUnlock()

	u.Lock()
	defer u.Unlock()

	if res := u.res.Get(email); res != nil {
		log.Debugf("user:%v, resource cache:%v", email, res.([]int64))
		return res.([]int64), nil
	}

	res, err := rbacClient.GetUserResourceIDs(email)
	if err != nil {
		return nil, errors.Trace(err)
	}
	u.res.Add(email, res)
	log.Debugf("user:%v, res:%v", email, res)
	return res, nil
}

// setResource 设置用户允许使用的资源列表.
func (u *userinfo) setResource(res []int64) {
	if u.Res = res; len(res) == 0 {
		return
	}

	buf := bytes.NewBufferString("")
	for _, id := range res {
		fmt.Fprintf(buf, "%d,", id)
	}
	buf.Truncate(buf.Len() - 1)
	u.ResKey = buf.String()
}

// setRoles 设置用户角色列表.
func (u *userinfo) setRoles(roles []int64) {
	if u.Roles = roles; len(roles) == 0 {
		return
	}

	buf := bytes.NewBufferString("")
	for _, id := range roles {
		fmt.Fprintf(buf, "%d,", id)
	}
	buf.Truncate(buf.Len() - 1)
	u.RolesKey = buf.String()
}

//validate 权限验证
func (u *userinfo) assert(resID int64) error {
	if u.IsAdmin {
		return nil
	}
	for _, id := range u.Res {
		if resID == id {
			return nil
		}
	}
	log.Errorf("account:%+v, resourceID:%d", *u, resID)
	return fmt.Errorf("you don't have permission to access")
}

//loadUserID 查找用户ID.
func (u *userDB) loadUser(email string) (meta.User, error) {
	u.Lock()
	defer u.Unlock()

	if cu := u.users.Get(email); cu != nil {
		user := cu.(*meta.User)
		log.Debugf("email:%v, user cache:%v", email, cu)
		return *user, nil
	}

	user, err := rbacClient.GetUser(email)
	if err != nil {
		return meta.User{}, errors.Trace(err)
	}

	u.users.Add(email, &user)

	log.Debugf("rbac email:%s, user:%+v", email, user)

	return user, nil
}

//loadRoles 查找用户所有角色.
func (u *userDB) loadRoles(email string) ([]int64, error) {
	u.RLock()
	if roles := u.roles.Get(email); roles != nil {
		u.RUnlock()
		log.Debugf("user:%v, roles cache:%v", email, roles.([]int64))
		return roles.([]int64), nil
	}
	u.RUnlock()

	u.Lock()
	defer u.Unlock()

	if roles := u.roles.Get(email); roles != nil {
		log.Debugf("user:%v, roles cache:%v", email, roles.([]int64))
		return roles.([]int64), nil
	}

	rs, err := rbacClient.GetUserRoles(email)
	if err != nil {
		return nil, errors.Trace(err)
	}

	var ids []int64
	for _, r := range rs {
		ids = append(ids, r.RoleID)
	}
	log.Debugf("email:%v, roles:%v", email, ids)

	u.roles.Add(email, ids)

	return ids, nil
}
