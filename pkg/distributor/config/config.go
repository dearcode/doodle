package config

import (
	"flag"

	dcfg "dearcode.net/crab/config"
	"dearcode.net/crab/orm"
)

type etcdConfig struct {
	Hosts string
}

type managerConfig struct {
	URL string
}

type serverConfig struct {
	SecretKey string
	BuildPath string
	Script    string
	Timeout   int
}

type Config struct {
	DB      orm.DB
	ETCD    etcdConfig
	Server  serverConfig
	Manager managerConfig
}

var (
	Distributor Config
	cfgPath     = flag.String("c", "./config/manager.ini", "config file")
)

func Load() error {
	return dcfg.LoadConfig(*cfgPath, &Distributor)
}
