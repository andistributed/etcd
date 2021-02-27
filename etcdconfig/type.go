package etcdconfig

import "github.com/coreos/etcd/clientv3"

type Configer func(*clientv3.Config)
