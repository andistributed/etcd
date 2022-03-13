package etcdlock_test

import (
	"sync"
	"testing"
	"time"

	"github.com/andistributed/etcd/etcdlock"
	"github.com/stretchr/testify/assert"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func TestLock(t *testing.T) {
	var counter int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	exec := func(i int) {
		lk, err := etcdlock.New(clientv3.Config{
			Endpoints: []string{"127.0.0.1:2379"},
		})
		if err != nil {
			panic(err)
		}
		defer lk.Close()

		var j int
		// LOOP:
		err = lk.TryLock(`test.lock.1`)
		if err != nil {
			t.Logf("[%d.%d]locked: %v", i, j, err)
			// if j < 5 {
			// 	time.Sleep(500 * time.Millisecond)
			// 	j++
			// 	goto LOOP
			// }
		} else {
			lk.Unlock()

			mu.Lock()
			counter++
			t.Logf("[%d]free: %v", i, counter)
			mu.Unlock()
		}
	}
	exec(-1)
	exec(-2)
	assert.Equal(t, int64(2), counter)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			time.Sleep(15 * time.Millisecond)
			defer wg.Done()
			exec(i)
		}(i)
	}
	wg.Wait()
	newCounter := counter + 1
	exec(-3)
	assert.Equal(t, counter, newCounter)
	assert.NotEqual(t, int64(13), counter)
}
