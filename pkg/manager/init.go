package manager

import (
	"dearcode.net/crab/http/client"
	"dearcode.net/crab/http/server"
	"dearcode.net/crab/log"
	"dearcode.net/crab/orm"

	"dearcode.net/doodle/pkg/manager/config"
	"dearcode.net/doodle/pkg/util/rbac"
)

var (
	rbacClient *rbac.Client
	mdb        *orm.DB
	httpClient *client.HTTPClient
)

// ServerInit 初始化HTTP接口.
func ServerInit() error {
	if err := config.Load(); err != nil {
		return err
	}
	mdb = &config.Manager.DB

	rbacClient = rbac.New(config.Manager.RBAC.Host, config.Manager.RBAC.Token)

	httpClient = client.New().SetLogger(log.GetLogger())

	server.RegisterPathMust(&serverCfg{}, "/config")
	server.RegisterPathMust(&account{}, "/account")

	server.RegisterPrefixMust(&debug{}, "/debug/pprof/")
	server.RegisterPrefixMust(&static{}, "/static/")
	server.RegisterPrefixMust(&static{}, "/")

	server.RegisterPathMust(&resource{}, "/resource/")
	server.RegisterPathMust(&resourceInfo{}, "/resource/info")
	server.RegisterPathMust(&resourceRole{}, "/resource/role/")

	server.RegisterPathMust(&cluster{}, "/cluster/")
	server.RegisterPathMust(&clusterInfo{}, "/cluster/info/")
	server.RegisterPathMust(&clusterNode{}, "/cluster/node/")

	server.RegisterPathMust(&role{}, "/role/")
	server.RegisterPathMust(&roleUser{}, "/role/user/")
	server.RegisterPathMust(&roleInfo{}, "/role/info/")
	server.RegisterPathMust(&userRole{}, "/user/role/")

	server.RegisterPathMust(&serviceInfo{}, "/service/info/")
	server.RegisterPathMust(&service{}, "/service/")

	server.RegisterPathMust(&nodes{}, "/nodes/")

	server.RegisterPathMust(&interfaceAction{}, "/interface/")
	server.RegisterPathMust(&interfaceRegister{}, "/interface/register/")
	server.RegisterPathMust(&interfaceRun{}, "/interface/run")
	server.RegisterPathMust(&interfaceInfo{}, "/interface/info")
	server.RegisterPathMust(&interfaceDeploy{}, "/interface/deploy")

	server.RegisterPathMust(&variableInfo{}, "/variable/infos")
	server.RegisterPathMust(&variable{}, "/variable/")

	server.RegisterPathMust(&appInfo{}, "/application/info")
	server.RegisterPathMust(&appInfos{}, "/application/infos")
	server.RegisterPathMust(&app{}, "/application/")

	server.RegisterPathMust(&relation{}, "/relation/")

	server.RegisterPathMust(&docs{}, "/docs/")

	server.RegisterPathMust(&statsSumAction{}, "/stats/sum/")
	server.RegisterPathMust(&statsTopApplication{}, "/stats/top/app/")
	server.RegisterPathMust(&statsTopInterface{}, "/stats/top/iface/")
	server.RegisterPathMust(&statsErrors{}, "/stats/error/")

	return nil
}
