package main

import (
	"encoding/binary"
	"flag"
	"fmt"

	"dearcode.net/crab/util/aes"

	rbacCfg "dearcode.net/doodle/rbac/config"
	rpCfg "dearcode.net/doodle/repeater/config"
)

var (
	decodeServiceKey = flag.String("decode_service_key", "", "decode service key.")
	appID            = flag.Int64("app_id", 0, "generate rbac app key.")
)

func parseServiceKey(key string) (int64, error) {
	if err := rpCfg.Load(); err != nil {
		return 0, nil
	}

	buf, err := aes.Decrypt(key, rpCfg.Repeater.Server.SecretKey)
	if err != nil {
		return 0, err
	}

	var id int64
	if _, err = fmt.Sscanf(string(buf), "%x.", &id); err != nil {
		return 0, err
	}

	return id, nil
}

func main() {
	flag.Parse()

	switch {
	case *decodeServiceKey != "":
		id, err := parseServiceKey(*decodeServiceKey)
		if err != nil {
			panic(err)
		}

		fmt.Printf("project:%v\n", id)
	case *appID != 0:
		if err := rbacCfg.Load(); err != nil {
			panic(err)
		}
		as := make([]byte, 8)
		binary.PutVarint(as, *appID)

		buf, err := aes.Encrypt(string(as), rbacCfg.RBAC.Server.Key)
		if err != nil {
			panic(err)
		}
		fmt.Printf("key:%v\n", buf)
	}
}
