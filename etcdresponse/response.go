package etcdresponse

import (
	"context"

	"github.com/andistributed/etcd/etcdevent"
	"github.com/coreos/etcd/clientv3"
)

// TxResponse 事务响应
type TxResponse struct {
	Success bool
	LeaseID clientv3.LeaseID
	Lease   clientv3.Lease
	Key     string
	Value   string
}

// WatchKeyChangeResponse 监听key 变化响应
type WatchKeyChangeResponse struct {
	Event      chan *etcdevent.KeyChangeEvent
	CancelFunc context.CancelFunc
	Watcher    clientv3.Watcher
}
