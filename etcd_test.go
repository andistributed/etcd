package etcd

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func InitEtcd() *Etcd {

	etcd, err := New([]string{"127.0.0.1:2379"}, time.Second*5)
	if err != nil {
		log.Fatal(err)
	}

	return etcd
}

func TestEtcd_Put(t *testing.T) {
	etcd := InitEtcd()

	err := etcd.Put("/echo", "echo-value")
	if err != nil {
		t.Error(err)
	}

	err = etcd.Put("/echo/one", "echo-value-one")
	if err != nil {
		t.Error(err)
	}

}
func TestEtcd_Get(t *testing.T) {

	etcd := InitEtcd()

	value, err := etcd.Get("/echo")
	if err != nil {

		t.Error(err)
	}

	log.Println("get a value:", string(value))
}

func TestEtcd_GetWithPrefixKey(t *testing.T) {
	etcd := InitEtcd()

	keys, values, err := etcd.GetWithPrefixKey("/echo")
	if err != nil {

		t.Error(err)
	}

	for i, key := range keys {

		t.Log("key:", string(key))
		t.Log("value:", string(values[i]))
	}

}

func TestEtcd_PutNotExist(t *testing.T) {

	etcd := InitEtcd()

	success, old, err := etcd.PutNotExist("/echo", "echo-value")
	if err != nil {
		t.Log(err)
	}

	t.Log("success", success)

	t.Log("old", string(old))
}

func TestEtcd_Update(t *testing.T) {

	etcd := InitEtcd()

	value, err := etcd.Get("/echo")
	if err != nil {
		t.Error(err)
	}

	success, err := etcd.Update("/echo", "echo-2", string(value))
	if err != nil {
		t.Error(err)
	}

	log.Println("success:", success)
}

func TestEtcd_Delete(t *testing.T) {

	etcd := InitEtcd()

	err := etcd.Delete("/echo")
	if err != nil {
		t.Error(err)
	}

}

func TestEtcd_DeleteWithPrefixKey(t *testing.T) {

	etcd := InitEtcd()

	err := etcd.DeleteWithPrefixKey("/echo")
	if err != nil {
		t.Error(err)
	}

}

func TestEtcd_Watch(t *testing.T) {

	etcd := InitEtcd()
	err := etcd.Put("/echo", "echo-value")
	if err != nil {
		t.Error(err)
	}

	g := &sync.WaitGroup{}
	g.Add(3)
	keyChangeEventResponse := etcd.Watch("/watch")
	go func() {
		err := etcd.Put("/watch", "watch-value")
		if err != nil {
			t.Error(err)
		}

		g.Done()

	}()

	go func() {
		err := etcd.Put("/watch", "watch-value")
		if err != nil {
			t.Error(err)
		}

		g.Done()

	}()

	go func() {
		err := etcd.Delete("/watch")
		if err != nil {
			t.Error(err)
		}

		g.Done()

	}()

	t.Log(<-keyChangeEventResponse.Event)
	t.Log(<-keyChangeEventResponse.Event)
	t.Log(<-keyChangeEventResponse.Event)

	g.Wait()
	_ = keyChangeEventResponse.Watcher.Close()

}

//
func TestEtcd_TxWithTTL(t *testing.T) {

	etcd := InitEtcd()

	txResponse, err := etcd.TxWithTTL("/ttl", "ttl", 10)
	if err != nil {
		t.Error(err)
	}

	t.Log("success:", txResponse.Success)

	if !txResponse.Success {
		t.Log(txResponse.Value)
		t.Log(txResponse.Key)

	}
}

func TestEtcd_TxKeepaliveWithTTL(t *testing.T) {

	etcd := InitEtcd()

	txResponse, err := etcd.TxKeepaliveWithTTL("/keep", "keep", 10)
	if err != nil {
		t.Error(err)
	}

	t.Log("success:", txResponse.Success)

	if !txResponse.Success {
		t.Log(txResponse.Value)
		t.Log(txResponse.Key)

	}

	_ = txResponse.Lease.Close()

	time.Sleep(time.Second * 3)
}

func TestEtcd_GetWithPrefixKeyChunk(t *testing.T) {
	etcd := InitEtcd()
	prefixKey := "/test/big-data/"
	for i := 0; i < 100; i++ {
		iStr := strconv.Itoa(i)
		_, _, err := etcd.PutNotExist(prefixKey+iStr, "echo-value-"+iStr)
		if err != nil {
			t.Error(err)
		}
	}
	keys, values, err := etcd.GetWithPrefixKeyLimit("/test/big-data/", 100)
	if err != nil {
		t.Error(err)
	}
	size := len(keys)
	vSize := len(values)
	assert.Equal(t, size, vSize)
	assert.Equal(t, size, 100)

	var i int
	etcd.GetWithPrefixKeyChunk(prefixKey, 21, func(key, value []byte) error {
		i++
		fmt.Printf("%d. %v  =================> %v\n", i, string(key), string(value))
		return nil
	})
	assert.Equal(t, size, i)

	/* not suppored -------------------
	i = 0
	etcd.GetWithPrefixKeyChunk(prefixKey, 21, func(key, value []byte) error {
		i++
		fmt.Printf("%d. %v  =================> %v\n", i, string(key), string(value))
		return nil
	}, clientv3.SortDescend)
	assert.Equal(t, size, i)
	*/
}
