package service

import (
	"context"
	"testing"
	"time"
)

func TestOptimizedMemoryPool(t *testing.T) {
	ctx := context.Background()
	pool := NewOptimizedMemoryPool(ctx)
	defer pool.Stop()

	pool.Start()

	sizes := []int{64, 256, 1024, 4096}
	for _, size := range sizes {
		buf := pool.Get(size)
		if len(buf) != size {
			t.Errorf("Get(%d) returned buffer with length %d", size, len(buf))
		}
		if cap(buf) < size {
			t.Errorf("Get(%d) returned buffer with capacity %d < %d", size, cap(buf), size)
		}
		pool.Put(buf)
	}
}

func TestOptimizedMemoryPoolStats(t *testing.T) {
	ctx := context.Background()
	pool := NewOptimizedMemoryPool(ctx)
	defer pool.Stop()

	pool.Start()

	buf := pool.Get(1024)
	pool.Put(buf)

	stats := pool.GetStats()
	if stats == nil {
		t.Error("GetStats() returned nil")
	}

	if stats["total_allocations"].(int64) < 1 {
		t.Error("total_allocations should be >= 1")
	}
}

func TestTypedObjectPool(t *testing.T) {
	factory := func() []int {
		return make([]int, 10)
	}
	reset := func(arr []int) {
		for i := range arr {
			arr[i] = 0
		}
	}

	pool := NewTypedObjectPool(factory, reset)

	item1 := pool.Get()
	item1[0] = 42
	pool.Put(item1)

	item2 := pool.Get()
	if item2[0] != 0 {
		t.Errorf("Expected item2[0] to be 0 after reset, got %d", item2[0])
	}
}

func TestTypedObjectPoolStats(t *testing.T) {
	factory := func() string {
		return ""
	}

	pool := NewTypedObjectPool(factory, nil)

	for i := 0; i < 100; i++ {
		item := pool.Get()
		pool.Put(item)
	}

	stats := pool.GetStats()
	if stats["total_gets"].(int64) != 100 {
		t.Errorf("Expected 100 total_gets, got %d", stats["total_gets"])
	}
}

func TestSlabPool(t *testing.T) {
	pool := NewSlabPool(64, 4096, 65536)

	buf := pool.Allocate(100)
	if cap(buf) < 64 {
		t.Errorf("Expected capacity >= 64, got %d", cap(buf))
	}

	pool.Release(buf)

	stats := pool.GetStats()
	if stats == nil {
		t.Error("GetStats() returned nil")
	}
}

func TestLockFreeQueue(t *testing.T) {
	q := NewLockFreeQueue[int]()
	q.Init()

	for i := 0; i < 100; i++ {
		if !q.Enqueue(i) {
			t.Errorf("Failed to enqueue %d", i)
		}
	}

	for i := 0; i < 100; i++ {
		val, ok := q.Dequeue()
		if !ok {
			t.Error("Dequeue returned false")
		}
		if val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	if !q.IsEmpty() {
		t.Error("Queue should be empty")
	}
}

func TestLockFreeStack(t *testing.T) {
	s := NewLockFreeStack[int]()

	for i := 0; i < 100; i++ {
		s.Push(i)
	}

	for i := 99; i >= 0; i-- {
		val, ok := s.Pop()
		if !ok {
			t.Error("Pop returned false")
		}
		if val != i {
			t.Errorf("Expected %d, got %d", i, val)
		}
	}

	if !s.IsEmpty() {
		t.Error("Stack should be empty")
	}
}

func TestLockFreeMap(t *testing.T) {
	m := NewLockFreeMap[string, int](16)

	m.Set("key1", 100)
	m.Set("key2", 200)

	val, ok := m.Get("key1")
	if !ok {
		t.Error("Get returned false for existing key")
	}
	if val != 100 {
		t.Errorf("Expected 100, got %d", val)
	}

	m.Delete("key1")
	_, ok = m.Get("key1")
	if ok {
		t.Error("Get should return false after delete")
	}
}

func TestLockFreeSet(t *testing.T) {
	s := NewLockFreeSet[int]()

	s.Add(1)
	s.Add(2)
	s.Add(1)

	if s.Size() != 2 {
		t.Errorf("Expected size 2, got %d", s.Size())
	}

	if !s.Contains(1) {
		t.Error("Set should contain 1")
	}

	if s.Contains(3) {
		t.Error("Set should not contain 3")
	}
}

func TestLockFreeRingBuffer(t *testing.T) {
	rb := NewLockFreeRingBuffer[string](5)

	for i := 0; i < 5; i++ {
		if !rb.Push("item") {
			t.Error("Push should succeed when buffer is not full")
		}
	}

	if !rb.IsFull() {
		t.Error("Buffer should be full")
	}

	if rb.Push("extra") {
		t.Error("Push should fail when buffer is full")
	}

	for i := 0; i < 5; i++ {
		val, ok := rb.Pop()
		if !ok {
			t.Error("Pop should succeed")
		}
		_ = val
	}

	if !rb.IsEmpty() {
		t.Error("Buffer should be empty")
	}
}

func TestLockFreeCounter(t *testing.T) {
	c := NewLockFreeCounter()

	c.Increment(10)
	if c.Get() != 10 {
		t.Errorf("Expected 10, got %d", c.Get())
	}

	c.Increment(5)
	if c.Get() != 15 {
		t.Errorf("Expected 15, got %d", c.Get())
	}

	c.Decrement(3)
	if c.Get() != 12 {
		t.Errorf("Expected 12, got %d", c.Get())
	}

	if c.GetMin() != 0 {
		t.Errorf("Expected min 0, got %d", c.GetMin())
	}
}

func TestLockFreeBitmap(t *testing.T) {
	bm := NewLockFreeBitmap(128)

	for i := 0; i < 128; i++ {
		if bm.Test(i) {
			t.Errorf("Bit %d should not be set initially", i)
		}
	}

	for i := 0; i < 64; i++ {
		if !bm.Set(i * 2) {
			t.Errorf("Failed to set bit %d", i*2)
		}
	}

	for i := 0; i < 128; i++ {
		expected := i%2 == 0
		if bm.Test(i) != expected {
			t.Errorf("Bit %d test failed", i)
		}
	}

	bm.Clear(0)
	if bm.Test(0) {
		t.Error("Bit 0 should be cleared")
	}
}

func TestLockFreeDeque(t *testing.T) {
	d := NewLockFreeDeque[string](5)

	for i := 0; i < 5; i++ {
		if !d.PushRight("item") {
			t.Error("PushRight should succeed")
		}
	}

	if !d.IsFull() {
		t.Error("Deque should be full")
	}

	if !d.PushLeft("extra") {
		t.Error("PushLeft should succeed (wrap around)")
	}

	item, ok := d.PopLeft()
	if !ok || item != "extra" {
		t.Error("PopLeft should return 'extra'")
	}

	item, ok = d.PopRight()
	if !ok {
		t.Error("PopRight should succeed")
	}

	d.PopLeft()
	d.PopLeft()
	d.PopLeft()
	d.PopLeft()

	if !d.IsEmpty() {
		t.Error("Deque should be empty")
	}
}

func TestExtremeQPSOptimizer(t *testing.T) {
	config := &QPSConfig{
		TargetQPS:   15000,
		WorkerCount: 4,
		BatchSize:   10,
	}

	optimizer := NewExtremeQPSOptimizer(config)
	defer optimizer.Stop()

	if err := optimizer.Start(); err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	optimizer.SetHandler(func(req *Request) *Response {
		return &Response{
			StatusCode: 200,
			Body:       []byte("OK"),
		}
	})

	req := &Request{
		ID:   "test-1",
		Path: "/api/test",
	}

	resp := optimizer.Handle(req)
	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestExtremeQPSOptimizerStats(t *testing.T) {
	config := &QPSConfig{
		TargetQPS:   15000,
		WorkerCount: 4,
	}

	optimizer := NewExtremeQPSOptimizer(config)
	defer optimizer.Stop()

	optimizer.Start()

	optimizer.SetHandler(func(req *Request) *Response {
		return &Response{StatusCode: 200}
	})

	for i := 0; i < 100; i++ {
		req := &Request{ID: "test"}
		optimizer.Handle(req)
	}

	stats := optimizer.GetStats()
	if stats["total_requests"].(int64) < 100 {
		t.Errorf("Expected at least 100 total_requests, got %d", stats["total_requests"])
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Error("Circuit breaker should be open after threshold failures")
	}

	time.Sleep(150 * time.Millisecond)

	cb.IsOpen()
}

func TestRateLimiterV2(t *testing.T) {
	rl := NewRateLimiterV2(100)

	allowed := 0
	for i := 0; i < 150; i++ {
		if rl.Allow() {
			allowed++
		}
	}

	if allowed < 100 {
		t.Errorf("Expected at least 100 allowed, got %d", allowed)
	}
}

func TestHeterogeneousComputer(t *testing.T) {
	hc := NewHeterogeneousComputer()
	defer hc.Stop()

	if err := hc.Start(); err != nil {
		t.Errorf("Start() failed: %v", err)
	}

	tensor := &Tensor{
		Shape: []int{10},
		Data:  []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	kernel := &Kernel{
		Name:   "test",
		Code:   []byte{},
		Inputs: []Tensor{*tensor},
		Output: tensor,
	}

	task := &ComputeTask{
		ID:     "test-1",
		Kernel: kernel,
		Device: DeviceCPU,
	}

	result := hc.ExecuteTask(task)
	if result == nil {
		t.Error("ExecuteTask returned nil")
	}
}

func TestHeterogeneousComputerStats(t *testing.T) {
	hc := NewHeterogeneousComputer()
	defer hc.Stop()

	hc.Start()

	tensor := &Tensor{
		Shape: []int{10},
		Data:  []float32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	}

	kernel := &Kernel{
		Name:   "test",
		Code:   []byte{},
		Inputs: []Tensor{*tensor},
		Output: tensor,
	}

	for i := 0; i < 10; i++ {
		task := &ComputeTask{
			ID:     "test",
			Kernel: kernel,
			Device: DeviceCPU,
		}
		hc.ExecuteTask(task)
	}

	stats := hc.GetStats()
	if stats["total_tasks"].(int64) < 10 {
		t.Errorf("Expected at least 10 total_tasks, got %d", stats["total_tasks"])
	}
}

func TestSIMDProcessor(t *testing.T) {
	simd := NewSIMDProcessor()

	a := []float32{1, 2, 3, 4}
	b := []float32{5, 6, 7, 8}

	result := simd.Add(a, b)
	expected := []float32{6, 8, 10, 12}

	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Add failed at index %d: expected %f, got %f", i, expected[i], result[i])
		}
	}

	result = simd.Mul(a, b)
	expected = []float32{5, 12, 21, 32}

	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Mul failed at index %d: expected %f, got %f", i, expected[i], result[i])
		}
	}
}

func TestVectorMath(t *testing.T) {
	vm := NewVectorMath()

	a := []float32{1, 2, 3}
	b := []float32{4, 5, 6}

	dot := vm.Dot(a, b)
	expected := float32(32)
	if dot != expected {
		t.Errorf("Dot failed: expected %f, got %f", expected, dot)
	}

	norm := vm.Norm(a)
	expectedNorm := float32(3.741657)
	if abs(norm-expectedNorm) > 0.001 {
		t.Errorf("Norm failed: expected %f, got %f", expectedNorm, norm)
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func TestWASMCryptoEngineV3(t *testing.T) {
	engine := NewWASMCryptoEngineV3("test-secret-key")

	plaintext := []byte("Hello, World!")
	ciphertext, err := engine.Encrypt(plaintext)
	if err != nil {
		t.Errorf("Encrypt failed: %v", err)
	}

	decrypted, err := engine.Decrypt(ciphertext)
	if err != nil {
		t.Errorf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: expected %s, got %s", plaintext, decrypted)
	}
}

func TestWASMAIInference(t *testing.T) {
	inference := NewWASMAIInference()
	defer inference.Stop()

	if err := inference.Start(); err != nil {
		t.Errorf("Start failed: %v", err)
	}

	inputShape := []int{10}
	outputShape := []int{5}
	weights := make([]float32, 50)
	for i := range weights {
		weights[i] = float32(i % 10)
	}

	err := inference.LoadModel("test-model", "TestModel", "1.0", weights, inputShape, outputShape)
	if err != nil {
		t.Errorf("LoadModel failed: %v", err)
	}

	input := make([]float32, 10)
	for i := range input {
		input[i] = float32(i)
	}

	output, err := inference.Infer("test-model", input)
	if err != nil {
		t.Errorf("Infer failed: %v", err)
	}

	if len(output) != 5 {
		t.Errorf("Expected output length 5, got %d", len(output))
	}
}

func TestWASMSandboxSecurity(t *testing.T) {
	security := NewWASMSandboxSecurity()
	defer security.Stop()

	if err := security.Start(); err != nil {
		t.Errorf("Start failed: %v", err)
	}

	policy := &SandboxPolicy{
		ID:               "test-policy",
		Name:             "Test Policy",
		MaxMemory:        32 * 1024 * 1024,
		MaxExecutionTime: 100 * time.Millisecond,
	}

	err := security.CreatePolicy(policy)
	if err != nil {
		t.Errorf("CreatePolicy failed: %v", err)
	}

	retrieved, err := security.GetPolicy("test-policy")
	if err != nil {
		t.Errorf("GetPolicy failed: %v", err)
	}

	if retrieved.ID != "test-policy" {
		t.Errorf("Expected policy ID 'test-policy', got '%s'", retrieved.ID)
	}
}

func BenchmarkLockFreeQueue(b *testing.B) {
	q := NewLockFreeQueue[int]()
	q.Init()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(i)
	}

	for i := 0; i < b.N; i++ {
		q.Dequeue()
	}
}

func BenchmarkLockFreeStack(b *testing.B) {
	s := NewLockFreeStack[int]()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Push(i)
	}

	for i := 0; i < b.N; i++ {
		s.Pop()
	}
}

func BenchmarkOptimizedMemoryPool(b *testing.B) {
	ctx := context.Background()
	pool := NewOptimizedMemoryPool(ctx)
	defer pool.Stop()
	pool.Start()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := pool.Get(1024)
		pool.Put(buf)
	}
}

func BenchmarkWASMCryptoEngineV3(b *testing.B) {
	engine := NewWASMCryptoEngineV3("test-key")
	data := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ct, _ := engine.Encrypt(data)
		engine.Decrypt(ct)
	}
}
