package distributor

import (
	"testing"

	"dearcode.net/doodle/pkg/distributor/config"
)

func TestServiceKey(t *testing.T) {
	//TODO 这个测试不需要跑，以后把代码合并后再跑
	config.Distributor.Server.SecretKey = "1qaz@WSX"
	ds := []struct {
		key string
		id  int64
	}{
		{"dhJgJns2tfBFvWVWUSGBfm1dsYVXAtTlye7csKmSgZo=", 1},
		{"+61FUC7/V/QxeZzpXV37e3jDOXEcAN3TXwFavJ1Ek9E=", 1234},
	}

	p := &service{}
	for _, data := range ds {
		p.ID = data.id
		if key := p.key(); key != data.key {
			t.Fatalf("invalid key:%v, expect:%v, id:%v", key, data.key, data.id)
		}
	}
}
