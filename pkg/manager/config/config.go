package config

import (
	"flag"

	dcfg "dearcode.net/crab/config"
	"dearcode.net/crab/orm"
)

type etcdConfig struct {
	Hosts string
}

type cacheConfig struct {
	Timeout int
}

type ssoConfig struct {
	URL       string
	Key       string
	VerifyURL string
}

type rbacConfig struct {
	Host  string
	Token string
}

type serverConfig struct {
	SecretKey string
	BuildPath string
	Script    string
	Timeout   int
	Domain    string
	WebPath   string
}

type Config struct {
	DB     orm.DB
	ETCD   etcdConfig
	Server serverConfig
	Cache  cacheConfig
	RBAC   rbacConfig
	SSO    ssoConfig
}

var (
	Manager Config
	cfgPath = flag.String("c", "./config/manager.ini", "config file")
)

func Load() error {
	return dcfg.LoadConfig(*cfgPath, &Manager)
}
