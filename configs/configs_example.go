package configs

import (
	"sync"
	"time"
)

func main() {
	cs := New("123.4.56.78", 8888, "access_key_id_123", "access_key_secret_123")
	//cs.SetLogger(myLogger) // set your owner logger that implemented the interface: configs.Logger
	var wg sync.WaitGroup
	cf := &Config{
		Tenant: "tenantId_123",
		Group:  "group_123",
		DataId: "dataId123",
		OnChange: func(namespace, group, dataId string, data []byte) {
			// handle changes
		},
	}
	if err := cs.OnChange(&wg, cf); err != nil {
		panic(err)
	}

	// simulate app running
	time.Sleep(3 * time.Second)

	// stop and exit
	if err := cs.Stop(&wg); err != nil {
		panic(err)
	}
}
