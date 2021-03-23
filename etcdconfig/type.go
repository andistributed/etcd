package etcdconfig

import clientv3 "go.etcd.io/etcd/client/v3"

type Configer func(*clientv3.Config)
