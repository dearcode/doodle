package main

import (
	"encoding/binary"
	"flag"
	"fmt"

	"dearcode.net/crab/util/aes"
)

var (
	token = flag.String("token", "", "decode token.")
	appID = flag.Int64("app_id", 0, "generate rbac app key.")
	key   = flag.String("key", "", "secret key.")
)

func parseToken() (int64, error) {
	buf, err := aes.Decrypt(*token, *key)
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
	case *token != "":
		id, err := parseToken()
		if err != nil {
			panic(err)
		}

		fmt.Printf("project:%v\n", id)
	case *appID != 0:
		as := make([]byte, 8)
		binary.PutVarint(as, *appID)

		buf, err := aes.Encrypt(string(as), *key)
		if err != nil {
			panic(err)
		}
		fmt.Printf("key:%v\n", buf)
	}
}
