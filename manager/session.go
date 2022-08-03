package manager

import (
	"fmt"
	"net/http"

	"dearcode.net/crab/cache"
	"dearcode.net/crab/log"
	"github.com/juju/errors"

	"dearcode.net/doodle/manager/config"
)

const (
	//session会话超时30分钟
	sessionTimeout = 1800
)

type sessionCache struct {
	cache *cache.Cache
}

func newSession() *sessionCache {
	return &sessionCache{cache: cache.NewCache(sessionTimeout)}
}

func (s *sessionCache) getToken(r *http.Request) (string, error) {
	if token := r.URL.Query().Get("token"); token != "" {
		return token, nil
	}

	c, err := r.Cookie(config.Manager.SSO.Key)
	if err != nil {
		fmt.Printf("key:%v\n", config.Manager.SSO.Key)
		return "", errors.Annotatef(err, "key:%s", config.Manager.SSO.Key)
	}
	return c.Value, nil
}

//verify 调用sso接口验证token返回用户信息.
func (s *sessionCache) verify(r *http.Request, token string) (*userinfo, error) {
	resp := struct {
		Status  int
		Message string
		Data    userinfo
	}{}

	url := fmt.Sprintf("%s?token=%s", config.Manager.SSO.VerifyURL, token)
	log.Debugf("url:%v", url)
	if err := httpClient.GetJSON(url, nil, &resp); err != nil {
		return nil, errors.Trace(err)
	}

	if resp.Status != 0 {
		return nil, errors.New(resp.Message)
	}

	return &resp.Data, nil
}

func (u userinfo) String() string {
	return u.Email
}

//loadInfo 加载资源与角色信息.
func (u *userinfo) loadInfo() error {
	res, err := userdb.loadResource(u.Email)
	if err != nil {
		return errors.Trace(err)
	}
	u.setResource(res)

	roles, err := userdb.loadRoles(u.Email)
	if err != nil {
		return errors.Trace(err)
	}

	u.setRoles(roles)

	u.IsAdmin = userdb.isAdmin(u.Email)

	return nil
}

func (s *sessionCache) User(w http.ResponseWriter, r *http.Request) (*userinfo, error) {
	//如果url中带token字段，说明是rbac回调的
	if token := r.URL.Query().Get("token"); token != "" {
		cookie := http.Cookie{Name: config.Manager.SSO.Key, Value: token, Path: "/"}
		http.SetCookie(w, &cookie)
		vals := r.URL.Query()
		vals.Del("token")
		argv := vals.Encode()
		url := r.URL.Path
		if argv != "" {
			url += "?" + argv
		}
		w.Header().Add("Location", url)
		w.WriteHeader(http.StatusTemporaryRedirect)
		log.Infof("%v remove token:%v, location:%v", s, token, url)
		return nil, errors.Errorf("retry")
	}

	token, err := s.getToken(r)
	if err != nil {
		return nil, errors.Trace(err)
	}

	log.Debugf("token:%v", token)
	if val := s.cache.Get(token); val != nil {
		user := val.(*userinfo)
		log.Debugf("cache userinfo:%v", user)
		return user, user.loadInfo()
	}

	user, err := s.verify(r, token)
	if err != nil {
		return nil, errors.Trace(err)
	}

	cookie := http.Cookie{Name: config.Manager.SSO.Key, Value: token, Path: "/"}
	http.SetCookie(w, &cookie)

	log.Debugf("userinfo:%+v", user)

	if err = user.loadInfo(); err != nil {
		return nil, errors.Trace(err)
	}

	mu, err := userdb.loadUser(user.Email)
	if err != nil {
		return nil, errors.Trace(err)
	}
	user.UserID = mu.ID
	user.User = mu.Name

	s.cache.Add(token, user)

	return user, nil
}
