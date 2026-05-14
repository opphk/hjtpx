package pool

import (
	"sync"
	"sync/atomic"
)

type PoolConfig struct {
	InitialSize int
	MaxSize     int
	New         func() interface{}
	Reset       func(interface{})
}

type ManagedPool struct {
	config    PoolConfig
	objects   []interface{}
	mu        sync.Mutex
	cond      *sync.Cond
	size      int32
	maxSize   int32
	activeGet int64
	activePut int64
}

func NewManagedPool(config PoolConfig) *ManagedPool {
	if config.MaxSize <= 0 {
		config.MaxSize = 1000
	}
	if config.InitialSize < 0 {
		config.InitialSize = 0
	}

	pool := &ManagedPool{
		config:  config,
		objects: make([]interface{}, 0, config.MaxSize),
		maxSize: int32(config.MaxSize),
	}

	pool.cond = sync.NewCond(&pool.mu)

	for i := 0; i < config.InitialSize; i++ {
		if config.New != nil {
			pool.objects = append(pool.objects, config.New())
			atomic.AddInt32(&pool.size, 1)
		}
	}

	return pool
}

func (p *ManagedPool) Get() interface{} {
	p.mu.Lock()

	for len(p.objects) == 0 {
		if atomic.LoadInt32(&p.size) >= p.maxSize {
			p.mu.Unlock()
			if p.config.New != nil {
				return p.config.New()
			}
			return nil
		}
		p.cond.Wait()
	}

	obj := p.objects[len(p.objects)-1]
	p.objects = p.objects[:len(p.objects)-1]
	atomic.AddInt32(&p.size, -1)
	atomic.AddInt64(&p.activeGet, 1)
	p.mu.Unlock()

	return obj
}

func (p *ManagedPool) Put(obj interface{}) {
	if obj == nil {
		return
	}

	if p.config.Reset != nil {
		p.config.Reset(obj)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if atomic.LoadInt32(&p.size) >= p.maxSize {
		p.mu.Unlock()
		return
	}

	p.objects = append(p.objects, obj)
	atomic.AddInt32(&p.size, 1)
	atomic.AddInt64(&p.activePut, 1)
	p.cond.Signal()
}

func (p *ManagedPool) Size() int32 {
	return atomic.LoadInt32(&p.size)
}

func (p *ManagedPool) Stats() PoolStats {
	return PoolStats{
		CurrentSize: atomic.LoadInt32(&p.size),
		MaxSize:     p.maxSize,
		ActiveGet:   atomic.LoadInt64(&p.activeGet),
		ActivePut:   atomic.LoadInt64(&p.activePut),
		Utilization: float64(atomic.LoadInt32(&p.size)) / float64(p.maxSize),
	}
}

type PoolStats struct {
	CurrentSize int32
	MaxSize     int32
	ActiveGet   int64
	ActivePut   int64
	Utilization float64
}

type BytesPool struct {
	pool *ManagedPool
}

func NewBytesPool(minSize, maxSize int) *BytesPool {
	return &BytesPool{
		pool: NewManagedPool(PoolConfig{
			InitialSize: minSize,
			MaxSize:     maxSize,
			New: func() interface{} {
				return make([]byte, 0, 4096)
			},
			Reset: func(obj interface{}) {
				if b, ok := obj.([]byte); ok {
					clear(b)
				}
			},
		}),
	}
}

func (bp *BytesPool) Get() []byte {
	return bp.pool.Get().([]byte)
}

func (bp *BytesPool) Put(b []byte) {
	bp.pool.Put(b)
}

func (bp *BytesPool) Stats() PoolStats {
	return bp.pool.Stats()
}

type ImageBufferPool struct {
	pool *ManagedPool
}

func NewImageBufferPool(minSize, maxSize int) *ImageBufferPool {
	return &ImageBufferPool{
		pool: NewManagedPool(PoolConfig{
			InitialSize: minSize,
			MaxSize:     maxSize,
			New: func() interface{} {
				return &ImageBuffer{data: make([]byte, 0, 65536)}
			},
		}),
	}
}

func (ibp *ImageBufferPool) Get() *ImageBuffer {
	return ibp.pool.Get().(*ImageBuffer)
}

func (ibp *ImageBufferPool) Put(buf *ImageBuffer) {
	buf.Reset()
	ibp.pool.Put(buf)
}

func (ibp *ImageBufferPool) Stats() PoolStats {
	return ibp.pool.Stats()
}

type ImageBuffer struct {
	data   []byte
	offset int
}

func (ib *ImageBuffer) Write(p []byte) (n int, err error) {
	if ib.offset+len(p) > cap(ib.data) {
		newData := make([]byte, len(ib.data)*2)
		copy(newData, ib.data)
		ib.data = newData
	}
	n = copy(ib.data[ib.offset:], p)
	ib.offset += n
	if n < len(p) {
		ib.data = append(ib.data, p[n:]...)
		ib.offset = len(ib.data)
	}
	return len(p), nil
}

func (ib *ImageBuffer) Bytes() []byte {
	return ib.data[:ib.offset]
}

func (ib *ImageBuffer) Reset() {
	ib.offset = 0
	if len(ib.data) > 0 {
		clear(ib.data)
	}
}

func (ib *ImageBuffer) Cap() int {
	return cap(ib.data)
}

func (ib *ImageBuffer) Len() int {
	return ib.offset
}

type TaskResultPool struct {
	pool *ManagedPool
}

func NewTaskResultPool(minSize, maxSize int) *TaskResultPool {
	return &TaskResultPool{
		pool: NewManagedPool(PoolConfig{
			InitialSize: minSize,
			MaxSize:     maxSize,
			New: func() interface{} {
				return &TaskResult{data: make(map[string]interface{})}
			},
		}),
	}
}

func (trp *TaskResultPool) Get() *TaskResult {
	return trp.pool.Get().(*TaskResult)
}

func (trp *TaskResultPool) Put(tr *TaskResult) {
	tr.Reset()
	trp.pool.Put(tr)
}

func (trp *TaskResultPool) Stats() PoolStats {
	return trp.pool.Stats()
}

type TaskResult struct {
	data    map[string]interface{}
	success bool
	err     error
}

func (tr *TaskResult) Set(key string, value interface{}) {
	tr.data[key] = value
}

func (tr *TaskResult) Get(key string) interface{} {
	return tr.data[key]
}

func (tr *TaskResult) SetSuccess(success bool) {
	tr.success = success
}

func (tr *TaskResult) SetError(err error) {
	tr.err = err
}

func (tr *TaskResult) Success() bool {
	return tr.success
}

func (tr *TaskResult) Error() error {
	return tr.err
}

func (tr *TaskResult) Reset() {
	for k := range tr.data {
		delete(tr.data, k)
	}
	tr.success = false
	tr.err = nil
}

type ringPool struct {
	ring   []interface{}
	head   int32
	tail   int32
	count  int32
	mu     sync.Mutex
}

func newRingBuffer(size int) *ringPool {
	return &ringPool{
		ring: make([]interface{}, size),
	}
}

func (rb *ringPool) Push(obj interface{}) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.ring[rb.tail] = obj
	rb.tail = (rb.tail + 1) % int32(len(rb.ring))

	if rb.count < int32(len(rb.ring)) {
		rb.count++
	} else {
		rb.head = (rb.head + 1) % int32(len(rb.ring))
	}
}

func (rb *ringPool) Pop() (interface{}, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		return nil, false
	}

	obj := rb.ring[rb.head]
	rb.head = (rb.head + 1) % int32(len(rb.ring))
	rb.count--
	return obj, true
}

func (rb *ringPool) Len() int32 {
	return atomic.LoadInt32(&rb.count)
}

func (rb *ringPool) Cap() int {
	return len(rb.ring)
}

var defaultBytesPool = (*BytesPool)(nil)
var defaultOnce sync.Once

func GetDefaultBytesPool() *BytesPool {
	defaultOnce.Do(func() {
		defaultBytesPool = NewBytesPool(100, 1000)
	})
	return defaultBytesPool
}

type PoolRegistry struct {
	pools map[string]*ManagedPool
	mu    sync.RWMutex
}

var registry = &PoolRegistry{
	pools: make(map[string]*ManagedPool),
}

func RegisterPool(name string, pool *ManagedPool) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.pools[name] = pool
}

func GetPool(name string) (*ManagedPool, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	pool, ok := registry.pools[name]
	return pool, ok
}

func GetAllPoolStats() map[string]PoolStats {
	registry.mu.RLock()
	defer registry.mu.RUnlock()

	stats := make(map[string]PoolStats)
	for name, pool := range registry.pools {
		stats[name] = pool.Stats()
	}
	return stats
}
