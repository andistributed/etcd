package etcd

import (
	"context"
	"time"

	"github.com/admpub/log"
	"github.com/andistributed/etcd/etcdconfig"
	"github.com/andistributed/etcd/etcdevent"
	"github.com/andistributed/etcd/etcdresponse"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Etcd is etcd wrap
type Etcd struct {
	endpoints []string
	client    *clientv3.Client
	kv        clientv3.KV

	timeout time.Duration
}

// New create a etcd
func New(endpoints []string, timeout time.Duration, opts ...etcdconfig.Configer) (etcd *Etcd, err error) {
	var client *clientv3.Client

	conf := clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
	}
	for _, opt := range opts {
		opt(&conf)
	}
	if client, err = clientv3.New(conf); err != nil {
		return
	}
	etcd = &Etcd{
		endpoints: endpoints,
		client:    client,
		kv:        clientv3.NewKV(client),
		timeout:   timeout,
	}
	return
}

// Get get value from a key
func (etcd *Etcd) Get(key string) (value []byte, err error) {
	var getResponse *clientv3.GetResponse
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	if getResponse, err = etcd.kv.Get(ctx, key, clientv3.WithLimit(1)); err != nil {
		return
	}

	if len(getResponse.Kvs) == 0 {
		return
	}

	value = getResponse.Kvs[0].Value

	return
}

// GetWithPrefixKey get values from prefixKey
func (etcd *Etcd) GetWithPrefixKey(prefixKey string) (keys [][]byte, values [][]byte, err error) {
	var getResponse *clientv3.GetResponse
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	if getResponse, err = etcd.kv.Get(ctx, prefixKey, clientv3.WithPrefix()); err != nil {
		return
	}

	size := len(getResponse.Kvs)
	if size == 0 {
		return
	}

	keys = make([][]byte, size)
	values = make([][]byte, size)

	for i, v := range getResponse.Kvs {
		keys[i] = v.Key
		values[i] = v.Value
	}

	return
}

// GetWithPrefixKeyLimit get values from prefixKey limit
func (etcd *Etcd) GetWithPrefixKeyLimit(prefixKey string, limit int64) (keys [][]byte, values [][]byte, err error) {
	var getResponse *clientv3.GetResponse
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	if getResponse, err = etcd.kv.Get(ctx, prefixKey, clientv3.WithPrefix(), clientv3.WithLimit(limit)); err != nil {
		return
	}

	size := len(getResponse.Kvs)
	if size == 0 {
		return
	}

	keys = make([][]byte, size)
	values = make([][]byte, size)

	for i, v := range getResponse.Kvs {
		keys[i] = v.Key
		values[i] = v.Value
	}

	return
}

func (etcd *Etcd) GetWithPrefixKeyChunk(prefixKey string, chunkSize int64, callback func(key, value []byte) error, sorts ...clientv3.SortOrder) (err error) {
	var getResponse *clientv3.GetResponse
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	sort := clientv3.SortAscend
	if len(sorts) > 0 {
		sort = sorts[0]
	}
	if sort != clientv3.SortAscend {
		panic("etcd.GetWithPrefixKeyChunk only supports clientv3.SortAscend")
	}
	if getResponse, err = etcd.kv.Get(ctx, prefixKey,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, sort),
		clientv3.WithLimit(chunkSize)); err != nil {
		return
	}

	size := len(getResponse.Kvs)
	if size == 0 {
		return
	}
	cursor := *getResponse.Kvs[size-1]
	if getResponse.More {
		getResponse.Kvs = getResponse.Kvs[0 : size-1]
	}

	for _, v := range getResponse.Kvs {
		if err = callback(v.Key, v.Value); err != nil {
			return err
		}
	}

	if getResponse.More {
		return etcd.GetWithFromKeyChunk(string(cursor.Key), chunkSize, getResponse.Count-int64(len(getResponse.Kvs)), func(resp *clientv3.GetResponse) error {
			for _, v := range resp.Kvs {
				if err := callback(v.Key, v.Value); err != nil {
					return err
				}
			}
			return nil
		}, sorts...)
	}
	return
}

func (etcd *Etcd) GetWithFromKeyChunk(fromKey string, chunkSize int64, total int64, callback func(*clientv3.GetResponse) error, sorts ...clientv3.SortOrder) (err error) {
	if total <= 0 {
		return
	}
	var getResponse *clientv3.GetResponse
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	limit := chunkSize
	if limit > total {
		limit = total
	}
	sort := clientv3.SortAscend
	if len(sorts) > 0 {
		sort = sorts[0]
	}
	if sort != clientv3.SortAscend {
		panic("etcd.GetWithFromKeyChunk only supports clientv3.SortAscend")
	}
	if getResponse, err = etcd.kv.Get(ctx, fromKey,
		clientv3.WithFromKey(),
		clientv3.WithSort(clientv3.SortByKey, sort),
		clientv3.WithLimit(limit)); err != nil {
		return
	}

	size := len(getResponse.Kvs)
	if size == 0 {
		return
	}
	if size == 1 {
		return callback(getResponse)
	}

	allFound := int64(size) < chunkSize
	cursor := *getResponse.Kvs[size-1]
	if !allFound {
		getResponse.Kvs = getResponse.Kvs[0 : size-1]
	}

	err = callback(getResponse)
	if err != nil {
		return
	}

	if allFound || !getResponse.More {
		return
	}

	return etcd.GetWithFromKeyChunk(string(cursor.Key), chunkSize, total-int64(len(getResponse.Kvs)), callback, sorts...)
}

// Put put a key
func (etcd *Etcd) Put(key, value string) (err error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	_, err = etcd.kv.Put(ctx, key, value)
	return
}

// PutNotExist put a key not exist
func (etcd *Etcd) PutNotExist(key, value string) (success bool, oldValue []byte, err error) {
	var txnResponse *clientv3.TxnResponse
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	txn := etcd.client.Txn(ctx)

	txnResponse, err = txn.If(clientv3.Compare(clientv3.Version(key), "=", 0)).
		Then(clientv3.OpPut(key, value)).
		Else(clientv3.OpGet(key)).
		Commit()

	if err != nil {
		return
	}

	if txnResponse.Succeeded {
		success = true
	} else {
		oldValue = make([]byte, 0)
		oldValue = txnResponse.Responses[0].GetResponseRange().Kvs[0].Value
	}

	return
}

func (etcd *Etcd) Update(key, value, oldValue string) (success bool, err error) {
	var txnResponse *clientv3.TxnResponse

	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	txn := etcd.client.Txn(ctx)
	txnResponse, err = txn.If(clientv3.Compare(clientv3.Value(key), "=", oldValue)).
		Then(clientv3.OpPut(key, value)).
		Commit()

	if err != nil {
		return
	}

	if txnResponse.Succeeded {
		success = true
	}

	return
}

func (etcd *Etcd) Delete(key string) (err error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	_, err = etcd.kv.Delete(ctx, key)
	return
}

// DeleteWithPrefixKey delete the keys with prefix key
func (etcd *Etcd) DeleteWithPrefixKey(prefixKey string) (err error) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	_, err = etcd.kv.Delete(ctx, prefixKey, clientv3.WithPrefix())
	return
}

// Watch watch a key
func (etcd *Etcd) Watch(key string) (keyChangeEventResponse *etcdresponse.WatchKeyChangeResponse) {
	watcher := clientv3.NewWatcher(etcd.client)
	watchChans := watcher.Watch(context.Background(), key)

	keyChangeEventResponse = &etcdresponse.WatchKeyChangeResponse{
		Event:   make(chan *etcdevent.KeyChangeEvent, 250),
		Watcher: watcher,
	}

	go func() {
		for ch := range watchChans {
			if ch.Canceled {
				goto End
			}
			for _, event := range ch.Events {
				etcd.handleKeyChangeEvent(event, keyChangeEventResponse.Event)
			}
		}

	End:
		log.Warn("the watcher lose for key: ", key)
	}()
	return
}

// WatchWithPrefixKey watch with prefix key
func (etcd *Etcd) WatchWithPrefixKey(prefixKey string) (keyChangeEventResponse *etcdresponse.WatchKeyChangeResponse) {
	watcher := clientv3.NewWatcher(etcd.client)
	watchChans := watcher.Watch(context.Background(), prefixKey, clientv3.WithPrefix())
	keyChangeEventResponse = &etcdresponse.WatchKeyChangeResponse{
		Event:   make(chan *etcdevent.KeyChangeEvent, 250),
		Watcher: watcher,
	}
	go func() {
		for ch := range watchChans {
			if ch.Canceled {
				goto End
			}
			for _, event := range ch.Events {
				etcd.handleKeyChangeEvent(event, keyChangeEventResponse.Event)
			}
		}

	End:
		log.Warn("the watcher lose for prefixKey: ", prefixKey)
	}()
	return
}

// handle the key change event
func (etcd *Etcd) handleKeyChangeEvent(event *clientv3.Event, events chan *etcdevent.KeyChangeEvent) {
	changeEvent := &etcdevent.KeyChangeEvent{
		Key: string(event.Kv.Key),
	}
	switch event.Type {
	case mvccpb.PUT:
		if event.IsCreate() {
			changeEvent.Type = etcdevent.KeyCreateChangeEvent
		} else {
			changeEvent.Type = etcdevent.KeyUpdateChangeEvent
		}
		changeEvent.Value = event.Kv.Value

	case mvccpb.DELETE:
		changeEvent.Type = etcdevent.KeyDeleteChangeEvent
	}
	events <- changeEvent
}

func (etcd *Etcd) TxWithTTL(key, value string, ttl int64) (txResponse *etcdresponse.TxResponse, err error) {
	var (
		txnResponse *clientv3.TxnResponse
		leaseID     clientv3.LeaseID
		v           []byte
	)
	lease := clientv3.NewLease(etcd.client)
	grantResponse, err := lease.Grant(context.Background(), ttl)
	leaseID = grantResponse.ID

	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	txn := etcd.client.Txn(ctx)
	txnResponse, err = txn.If(
		clientv3.Compare(clientv3.Version(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(leaseID))).Commit()

	if err != nil {
		_ = lease.Close()
		return
	}

	txResponse = &etcdresponse.TxResponse{
		LeaseID: leaseID,
		Lease:   lease,
	}
	if txnResponse.Succeeded {
		txResponse.Success = true
	} else {
		// close the lease
		_ = lease.Close()
		v, err = etcd.Get(key)
		if err != nil {
			return
		}
		txResponse.Success = false
		txResponse.Key = key
		txResponse.Value = string(v)
	}
	return
}

func (etcd *Etcd) TxKeepaliveWithTTL(key, value string, ttl int64) (txResponse *etcdresponse.TxResponse, err error) {
	var (
		txnResponse    *clientv3.TxnResponse
		leaseID        clientv3.LeaseID
		aliveResponses <-chan *clientv3.LeaseKeepAliveResponse
		v              []byte
	)
	lease := clientv3.NewLease(etcd.client)
	grantResponse, err := lease.Grant(context.Background(), ttl)
	leaseID = grantResponse.ID

	if aliveResponses, err = lease.KeepAlive(context.Background(), leaseID); err != nil {
		return
	}

	go func() {
		for ch := range aliveResponses {
			if ch == nil {
				goto End
			}
		}

	End:
		log.Warnf("the tx keepalive has lose key: %s", key)
	}()

	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	txn := etcd.client.Txn(ctx)
	txnResponse, err = txn.If(
		clientv3.Compare(clientv3.Version(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(leaseID))).
		Else(
			clientv3.OpGet(key),
		).Commit()

	if err != nil {
		_ = lease.Close()
		return
	}

	txResponse = &etcdresponse.TxResponse{
		LeaseID: leaseID,
		Lease:   lease,
	}
	if txnResponse.Succeeded {
		txResponse.Success = true
	} else {
		// close the lease
		_ = lease.Close()
		txResponse.Success = false
		if v, err = etcd.Get(key); err != nil {
			return
		}
		txResponse.Key = key
		txResponse.Value = string(v)
	}
	return
}

func (etcd *Etcd) TxKeepaliveWithTTLAndChan(key, value string, ttl int64) (txResponse *etcdresponse.TxResponseWithChan, err error) {
	var (
		txnResponse    *clientv3.TxnResponse
		leaseID        clientv3.LeaseID
		aliveResponses <-chan *clientv3.LeaseKeepAliveResponse
		v              []byte
		grantResponse  *clientv3.LeaseGrantResponse
	)
	lease := clientv3.NewLease(etcd.client)
	grantResponse, err = lease.Grant(context.Background(), ttl)
	if err != nil {
		return
	}
	leaseID = grantResponse.ID

	if aliveResponses, err = lease.KeepAlive(context.Background(), leaseID); err != nil {
		return
	}

	txResponse = &etcdresponse.TxResponseWithChan{
		LeaseID: leaseID,
		Lease:   lease,
	}

	go func() {
		for ch := range aliveResponses {
			if ch == nil {
				goto End
			}
		}

	End:
		log.Warnf("the tx keepalive has lose key <-----> %s", key)
		if txResponse.StateChan != nil {
			txResponse.StateChan <- false
		}
	}()

	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	txn := etcd.client.Txn(ctx)
	txnResponse, err = txn.If(
		clientv3.Compare(clientv3.Version(key), "=", 0)).
		Then(clientv3.OpPut(key, value, clientv3.WithLease(leaseID))).
		Else(
			clientv3.OpGet(key),
		).Commit()

	if err != nil {
		_ = lease.Close()
		return
	}

	if txnResponse.Succeeded {
		txResponse.Success = true
		txResponse.StateChan = make(chan bool)
	} else {
		// close the lease
		_ = lease.Close()
		txResponse.Success = false
		if v, err = etcd.Get(key); err != nil {
			return
		}
		txResponse.Key = key
		txResponse.Value = string(v)
	}
	return
}

// Transfer transfer from to with value
func (etcd *Etcd) Transfer(from string, to string, value string) (success bool, err error) {
	var txnResponse *clientv3.TxnResponse

	ctx, cancelFunc := context.WithTimeout(context.Background(), etcd.timeout)
	defer cancelFunc()

	txn := etcd.client.Txn(ctx)
	txnResponse, err = txn.If(
		clientv3.Compare(clientv3.Value(from), "=", value)).
		Then(
			clientv3.OpDelete(from),
			clientv3.OpPut(to, value),
		).Commit()

	if err != nil {
		return
	}
	success = txnResponse.Succeeded
	return
}
