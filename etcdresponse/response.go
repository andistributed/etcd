package etcdresponse

import (
	"context"

	"github.com/andistributed/etcd/etcdevent"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// TxResponse 事务响应
type TxResponse struct {
	Success bool
	LeaseID clientv3.LeaseID
	Lease   clientv3.Lease
	Key     string
	Value   string
}

// TxResponseWithChan 事务响应含管道
type TxResponseWithChan struct {
	Success   bool
	LeaseID   clientv3.LeaseID
	Lease     clientv3.Lease
	Key       string
	Value     string
	StateChan chan bool
}

// WatchKeyChangeResponse 监听key 变化响应
type WatchKeyChangeResponse struct {
	Event      chan *etcdevent.KeyChangeEvent
	CancelFunc context.CancelFunc
	Watcher    clientv3.Watcher
}
