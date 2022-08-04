package config

import (
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
)

func Load(path string) error {
	return dcfg.LoadConfig(path, &Distributor)
}
