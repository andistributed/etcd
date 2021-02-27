package etcdutils

import "errors"

// ErrNoLocalIPFound 没有找到本地IP
var ErrNoLocalIPFound = errors.New("no local ip found")

// ErrLockAlreadyOccupied 锁已被占用
var ErrLockAlreadyOccupied = errors.New("Lock is already occupied")
