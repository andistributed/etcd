package etcdevent

// KeyChangeEvent key 变化事件
type KeyChangeEvent struct {
	Type  int
	Key   string
	Value []byte
}

const (
	KeyCreateChangeEvent = iota
	KeyUpdateChangeEvent
	KeyDeleteChangeEvent
)
