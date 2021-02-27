package etcdlock

import (
	"context"

	"github.com/andistributed/etcd/etcdutils"
	"github.com/coreos/etcd/clientv3"
)

// DefaultDir 默认目录
var DefaultDir = `/andistributed/etcd/lock/`

// Lock 分布式锁(TXN事务)
type Lock struct {
	// etcd客户端
	kv    clientv3.KV
	lease clientv3.Lease

	jobName    string             // 任务名
	cancelFunc context.CancelFunc // 用于终止自动续租
	leaseID    clientv3.LeaseID   // 租约ID
	isLocked   bool               // 是否上锁成功
	dirName    string
}

// New 初始化一把锁
func New(jobName string, kv clientv3.KV, lease clientv3.Lease, dirNames ...string) (jobLock *Lock) {
	jobLock = &Lock{
		kv:      kv,
		lease:   lease,
		jobName: jobName,
		dirName: DefaultDir,
	}
	if len(dirNames) > 0 && len(dirNames[0]) > 0 {
		jobLock.dirName = dirNames[0]
	}
	return
}

// TryLock 尝试上锁
func (jobLock *Lock) TryLock() (err error) {
	var (
		leaseGrantResp *clientv3.LeaseGrantResponse
		cancelCtx      context.Context
		cancelFunc     context.CancelFunc
		leaseID        clientv3.LeaseID
		keepRespChan   <-chan *clientv3.LeaseKeepAliveResponse
		txn            clientv3.Txn
		lockKey        string
		txnResp        *clientv3.TxnResponse
		localIP        string
	)

	if localIP, err = etcdutils.GetLocalIP(); err != nil {
		localIP = "127.0.0.1"
	}

	// 1, 创建租约(5秒)
	if leaseGrantResp, err = jobLock.lease.Grant(context.TODO(), 5); err != nil {
		return
	}

	// context用于取消自动续租
	cancelCtx, cancelFunc = context.WithCancel(context.TODO())

	// 租约ID
	leaseID = leaseGrantResp.ID

	// 2, 自动续租
	if keepRespChan, err = jobLock.lease.KeepAlive(cancelCtx, leaseID); err != nil {
		goto FAIL
	}

	// 3, 处理续租应答的协程
	go func() {
		var (
			keepResp *clientv3.LeaseKeepAliveResponse
		)
		for {
			select {
			case keepResp = <-keepRespChan: // 自动续租的应答
				if keepResp == nil {
					return
				}
			}
		}
	}()

	// 4, 创建事务txn
	txn = jobLock.kv.Txn(context.TODO())

	// 锁路径
	lockKey = jobLock.dirName + localIP + "/" + jobLock.jobName

	// 5, 事务抢锁
	txn.If(clientv3.Compare(clientv3.CreateRevision(lockKey), "=", 0)).
		Then(clientv3.OpPut(lockKey, "", clientv3.WithLease(leaseID))).
		Else(clientv3.OpGet(lockKey))

	// 提交事务
	if txnResp, err = txn.Commit(); err != nil {
		goto FAIL
	}

	// 6, 成功返回, 失败释放租约
	if !txnResp.Succeeded { // 锁被占用
		err = etcdutils.ErrLockAlreadyOccupied
		goto FAIL
	}

	// 抢锁成功
	jobLock.leaseID = leaseID
	jobLock.cancelFunc = cancelFunc
	jobLock.isLocked = true
	return

FAIL:
	cancelFunc()                                  // 取消自动续租
	jobLock.lease.Revoke(context.TODO(), leaseID) //  释放租约
	return
}

// Unlock 释放锁
func (jobLock *Lock) Unlock() {
	if jobLock.isLocked {
		jobLock.cancelFunc()                                  // 取消我们程序自动续租的协程
		jobLock.lease.Revoke(context.TODO(), jobLock.leaseID) // 释放租约
	}
}
