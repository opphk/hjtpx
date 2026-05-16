# API性能优化指南

## 响应时间目标

| 接口 | 目标响应时间 | P99目标 |
|------|-------------|--------|
| 滑块验证码生成 | < 100ms | < 200ms |
| 点选验证码生成 | < 150ms | < 300ms |
| 验证码验证 | < 50ms | < 100ms |
| 统计数据查询 | < 200ms | < 500ms |
| 管理API | < 100ms | < 200ms |
| 健康检查 | < 10ms | < 20ms |

## 1. 图片生成优化

### 1.1 图片缓存

```go
type CaptchaCache struct {
    images sync.Map
    ttl    time.Duration
}

var globalCaptchaCache *CaptchaCache

func init() {
    globalCaptchaCache = &CaptchaCache{
        ttl: 5 * time.Minute,
    }
    go globalCaptchaCache.cleanup()
}

func (c *CaptchaCache) Get(key string) (*CaptchaImage, bool) {
    if val, ok := c.images.Load(key); ok {
        cached := val.(*cachedImage)
        if time.Since(cached.createdAt) < c.ttl {
            return cached.image, true
        }
        c.images.Delete(key)
    }
    return nil, false
}

func (c *CaptchaCache) Set(key string, img *CaptchaImage) {
    c.images.Store(key, &cachedImage{
        image:    img,
        createdAt: time.Now(),
    })
}

func (c *CaptchaCache) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        now := time.Now()
        c.images.Range(func(key, val interface{}) bool {
            cached := val.(*cachedImage)
            if now.Sub(cached.createdAt) > c.ttl {
                c.images.Delete(key)
            }
            return true
        })
    }
}
```

### 1.2 并发图片生成

```go
type ImageGeneratorPool struct {
    workers int
    jobs    chan ImageJob
    results chan ImageResult
}

type ImageJob struct {
    ID       string
    Type     string
    Response chan<- ImageResult
}

type ImageResult struct {
    Background string
    Slider     string
    TargetX    int
    TargetY    int
    Error      error
}

func NewImageGeneratorPool(workers int) *ImageGeneratorPool {
    pool := &ImageGeneratorPool{
        workers: workers,
        jobs:    make(chan ImageJob, 1000),
        results: make(chan ImageResult, 1000),
    }
    
    for i := 0; i < workers; i++ {
        go pool.worker()
    }
    
    return pool
}

func (p *ImageGeneratorPool) worker() {
    for job := range p.jobs {
        result := generateImage(job.Type)
        job.Response <- result
    }
}

func (p *ImageGeneratorPool) Generate(typ string) (ImageResult, error) {
    response := make(chan ImageResult, 1)
    p.jobs <- ImageJob{Type: typ, Response: response}
    result := <-response
    return result, result.Error
}
```

### 1.3 异步图片生成

```go
type AsyncImageGenerator struct {
    queue    chan ImageRequest
    workers  int
    wg       sync.WaitGroup
}

type ImageRequest struct {
    SessionID string
    Type      string
    Result    chan<- ImageResponse
}

type ImageResponse struct {
    Background string
    Slider     string
    TargetX    int
    TargetY    int
}

func NewAsyncImageGenerator(workers int, queueSize int) *AsyncImageGenerator {
    gen := &AsyncImageGenerator{
        queue:   make(chan ImageRequest, queueSize),
        workers: workers,
    }
    
    for i := 0; i < workers; i++ {
        gen.wg.Add(1)
        go gen.worker()
    }
    
    return gen
}

func (g *AsyncImageGenerator) worker() {
    defer g.wg.Done()
    for req := range g.queue {
        bg, slider, x, y := generateCaptchaImage(req.Type)
        req.Result <- ImageResponse{
            Background: bg,
            Slider:     slider,
            TargetX:    x,
            TargetY:    y,
        }
    }
}

func (g *AsyncImageGenerator) Generate(sessionID, typ string) ImageResponse {
    result := make(chan ImageResponse, 1)
    g.queue <- ImageRequest{
        SessionID: sessionID,
        Type:      typ,
        Result:    result,
    }
    return <-result
}
```

## 2. 数据库查询优化

### 2.1 预编译语句池

```go
type PreparedStatements struct {
    db          *sql.DB
    statements   map[string]*sql.Stmt
    mutex       sync.RWMutex
}

func NewPreparedStatements(db *sql.DB) *PreparedStatements {
    ps := &PreparedStatements{
        db:        db,
        statements: make(map[string]*sql.Stmt),
    }
    
    ps.prepare("get_verification", `
        SELECT id, session_id, captcha_type, status, risk_score, created_at
        FROM verifications
        WHERE session_id = ?`)
    
    ps.prepare("count_by_status", `
        SELECT COUNT(*) FROM verification_logs
        WHERE application_id = ? AND status = ?`)
    
    return ps
}

func (ps *PreparedStatements) Get(name string, args ...interface{}) (*sql.Row, error) {
    ps.mutex.RLock()
    stmt, ok := ps.statements[name]
    ps.mutex.RUnlock()
    
    if !ok {
        return nil, fmt.Errorf("statement not found: %s", name)
    }
    
    return stmt.QueryRow(args...), nil
}
```

### 2.2 查询结果缓存

```go
type QueryCache struct {
    cache   map[string]*CacheEntry
    mutex   sync.RWMutex
    maxSize int
    ttl     time.Duration
}

type CacheEntry struct {
    Data      interface{}
    CreatedAt time.Time
}

func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
    cache := &QueryCache{
        cache:   make(map[string]*CacheEntry),
        maxSize: maxSize,
        ttl:     ttl,
    }
    
    go cache.cleanup()
    return cache
}

func (c *QueryCache) Get(key string) (interface{}, bool) {
    c.mutex.RLock()
    defer c.mutex.RUnlock()
    
    if entry, ok := c.cache[key]; ok {
        if time.Since(entry.CreatedAt) < c.ttl {
            return entry.Data, true
        }
    }
    return nil, false
}

func (c *QueryCache) Set(key string, data interface{}) {
    c.mutex.Lock()
    defer c.mutex.Unlock()
    
    if len(c.cache) >= c.maxSize {
        c.evictOldest()
    }
    
    c.cache[key] = &CacheEntry{
        Data:      data,
        CreatedAt: time.Now(),
    }
}
```

### 2.3 批量查询优化

```go
func GetApplicationsByIDs(ids []uint) (map[uint]*Application, error) {
    if len(ids) == 0 {
        return make(map[uint]*Application), nil
    }
    
    result := make(map[uint]*Application, len(ids))
    
    chunks := chunkSlice(ids, 100)
    for _, chunk := range chunks {
        placeholders := make([]string, len(chunk))
        args := make([]interface{}, len(chunk))
        for i, id := range chunk {
            placeholders[i] = "?"
            args[i] = id
        }
        
        query := fmt.Sprintf(`
            SELECT id, name, user_id, api_key, is_active, created_at
            FROM applications
            WHERE id IN (%s)`, strings.Join(placeholders, ","))
        
        rows, err := db.Query(query, args...)
        if err != nil {
            return nil, err
        }
        
        for rows.Next() {
            var app Application
            rows.Scan(&app.ID, &app.Name, &app.UserID, &app.APIKey, &app.IsActive, &app.CreatedAt)
            result[app.ID] = &app
        }
        rows.Close()
    }
    
    return result, nil
}

func chunkSlice(slice []uint, chunkSize int) [][]uint {
    var chunks [][]uint
    for i := 0; i < len(slice); i += chunkSize {
        end := i + chunkSize
        if end > len(slice) {
            end = len(slice)
        }
        chunks = append(chunks, slice[i:end])
    }
    return chunks
}
```

## 3. Redis缓存优化

### 3.1 多级缓存

```go
type MultiLevelCache struct {
    l1 *sync.Map
    l2 *redis.Client
    ttl time.Duration
}

func NewMultiLevelCache(rdb *redis.Client, ttl time.Duration) *MultiLevelCache {
    return &MultiLevelCache{
        l1:  &sync.Map{},
        l2:  rdb,
        ttl: ttl,
    }
}

func (c *MultiLevelCache) Get(key string) (string, error) {
    if val, ok := c.l1.Load(key); ok {
        return val.(string), nil
    }
    
    val, err := c.l2.Get(context.Background(), key).Result()
    if err == nil {
        c.l1.Store(key, val)
    }
    
    return val, err
}

func (c *MultiLevelCache) Set(key string, val interface{}) error {
    c.l1.Store(key, val)
    return c.l2.Set(context.Background(), key, val, c.ttl).Err()
}
```

### 3.2 缓存预热

```go
func WarmupCache() error {
    ctx := context.Background()
    
    popularApps := []uint{1, 2, 3, 4, 5}
    
    var wg sync.WaitGroup
    for _, appID := range popularApps {
        wg.Add(1)
        go func(id uint) {
            defer wg.Done()
            
            stats, err := GetAppStats(id)
            if err != nil {
                return
            }
            
            cacheKey := fmt.Sprintf("stats:app:%d", id)
            data, _ := json.Marshal(stats)
            redis.Set(ctx, cacheKey, data, 5*time.Minute)
        }(appID)
    }
    
    wg.Wait()
    return nil
}
```

### 3.3 缓存失效策略

```go
type CacheInvalidation struct {
    patterns []string
    mutex    sync.RWMutex
}

func NewCacheInvalidation() *CacheInvalidation {
    ci := &CacheInvalidation{
        patterns: make([]string, 0),
    }
    go ci.listen()
    return ci
}

func (ci *CacheInvalidation) Register(pattern string) {
    ci.mutex.Lock()
    defer ci.mutex.Unlock()
    ci.patterns = append(ci.patterns, pattern)
}

func (ci *CacheInvalidation) Invalidate(key string) error {
    ctx := context.Background()
    
    ci.mutex.RLock()
    defer ci.mutex.RUnlock()
    
    for _, pattern := range ci.patterns {
        if strings.Contains(key, pattern) {
            keys, err := redis.Keys(ctx, pattern+"*").Result()
            if err != nil {
                return err
            }
            
            if len(keys) > 0 {
                redis.Del(ctx, keys...)
            }
        }
    }
    
    return redis.Del(ctx, key).Err()
}
```

## 4. 异步处理优化

### 4.1 异步日志写入

```go
type AsyncLogger struct {
    buffer   chan *LogEntry
    flushInt time.Duration
    db       *sql.DB
    wg       sync.WaitGroup
}

type LogEntry struct {
    VerificationID uint
    SessionID     string
    ApplicationID uint
    Status       string
    Duration     int64
}

func NewAsyncLogger(db *sql.DB, bufferSize int, flushInt time.Duration) *AsyncLogger {
    logger := &AsyncLogger{
        buffer:   make(chan *LogEntry, bufferSize),
        flushInt: flushInt,
        db:       db,
    }
    
    logger.wg.Add(1)
    go logger.worker()
    
    return logger
}

func (l *AsyncLogger) Log(entry *LogEntry) {
    select {
    case l.buffer <- entry:
    default:
        fmt.Println("Log buffer full, dropping entry")
    }
}

func (l *AsyncLogger) worker() {
    defer l.wg.Done()
    ticker := time.NewTicker(l.flushInt)
    
    var entries []*LogEntry
    
    for {
        select {
        case entry := <-l.buffer:
            entries = append(entries, entry)
            if len(entries) >= 100 {
                l.flush(entries)
                entries = entries[:0]
            }
        case <-ticker.C:
            if len(entries) > 0 {
                l.flush(entries)
                entries = entries[:0]
            }
        }
    }
}

func (l *AsyncLogger) flush(entries []*LogEntry) {
    if len(entries) == 0 {
        return
    }
    
    batch := make([]models.VerificationLog, len(entries))
    for i, e := range entries {
        batch[i] = models.VerificationLog{
            VerificationID: e.VerificationID,
            SessionID:     e.SessionID,
            ApplicationID: e.ApplicationID,
            Status:        e.Status,
            Duration:      e.Duration,
        }
    }
    
    db.CreateInBatches(batch, 50)
}
```

### 4.2 异步统计更新

```go
type AsyncStatsUpdater struct {
    updates chan StatsUpdate
    ticker  *time.Ticker
}

type StatsUpdate struct {
    AppID     uint
    Type      string
    Timestamp time.Time
}

func NewAsyncStatsUpdater() *AsyncStatsUpdater {
    updater := &AsyncStatsUpdater{
        updates: make(chan StatsUpdate, 1000),
        ticker:  time.NewTicker(1 * time.Minute),
    }
    
    go updater.run()
    return updater
}

func (u *AsyncStatsUpdater) Update(appID uint, typ string) {
    u.updates <- StatsUpdate{
        AppID:     appID,
        Type:      typ,
        Timestamp: time.Now(),
    }
}

func (u *AsyncStatsUpdater) run() {
    agg := make(map[uint]map[string]int64)
    
    for {
        select {
        case update := <-u.updates:
            if agg[update.AppID] == nil {
                agg[update.AppID] = make(map[string]int64)
            }
            agg[update.AppID][update.Type]++
            
        case <-u.ticker.C:
            u.persist(agg)
            agg = make(map[uint]map[string]int64)
        }
    }
}

func (u *AsyncStatsUpdater) persist(agg map[uint]map[string]int64) {
    for appID, stats := range agg {
        for typ, count := range stats {
            UpdateStatsInDB(appID, typ, count)
        }
    }
}
```

## 5. 并发控制

### 5.1 限流器

```go
type RateLimiter struct {
    requests map[string][]time.Time
    mutex   sync.RWMutex
    limit   int
    window  time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        requests: make(map[string][]time.Time),
        limit:    limit,
        window:   window,
    }
}

func (rl *RateLimiter) Allow(key string) bool {
    rl.mutex.Lock()
    defer rl.mutex.Unlock()
    
    now := time.Now()
    windowStart := now.Add(-rl.window)
    
    requests := rl.requests[key]
    valid := make([]time.Time, 0)
    
    for _, t := range requests {
        if t.After(windowStart) {
            valid = append(valid, t)
        }
    }
    
    if len(valid) >= rl.limit {
        rl.requests[key] = valid
        return false
    }
    
    valid = append(valid, now)
    rl.requests[key] = valid
    return true
}
```

### 5.2 连接池管理

```go
type WorkerPool struct {
    tasks    chan func()
    workers  int
    wg       sync.WaitGroup
    semaphore chan struct{}
}

func NewWorkerPool(workers, queueSize int) *WorkerPool {
    pool := &WorkerPool{
        tasks:    make(chan func(), queueSize),
        workers:  workers,
        semaphore: make(chan struct{}, workers),
    }
    
    for i := 0; i < workers; i++ {
        pool.wg.Add(1)
        go pool.worker()
    }
    
    return pool
}

func (p *WorkerPool) Submit(task func()) {
    p.tasks <- task
}

func (p *WorkerPool) worker() {
    defer p.wg.Done()
    for task := range p.tasks {
        p.semaphore <- struct{}{}
        go func(t func()) {
            defer func() { <-p.semaphore }()
            t()
        }(task)
    }
}

func (p *WorkerPool) Shutdown() {
    close(p.tasks)
    p.wg.Wait()
}
```

## 6. 性能监控

### 6.1 请求追踪

```go
type RequestTracer struct {
    tracer  otel.Tracer
    metrics *metrics.Client
}

func (tracer *RequestTracer) TraceHandler(name string, handler gin.HandlerFunc) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        
        span, ctx := tracer.tracer.Start(c.Request.Context(), name)
        c.Request = c.Request.WithContext(ctx)
        
        c.Next()
        
        duration := time.Since(start)
        span.End()
        
        tracer.metrics.RecordDuration(name, duration)
        tracer.metrics.IncrementCounter(name+".requests")
    }
}
```

### 6.2 性能指标

```go
type Metrics struct {
    requestsTotal   map[string]*prometheus.CounterVec
    requestDuration map[string]*prometheus.HistogramVec
    inFlight       map[string]*prometheus.GaugeVec
}

func NewMetrics() *Metrics {
    m := &Metrics{
        requestsTotal:   make(map[string]*prometheus.CounterVec),
        requestDuration: make(map[string]*prometheus.HistogramVec),
        inFlight:       make(map[string]*prometheus.GaugeVec),
    }
    return m
}

func (m *Metrics) RecordRequest(endpoint string, duration time.Duration) {
    m.requestsTotal[endpoint].WithLabelValues("success").Inc()
    m.requestDuration[endpoint].Observe(duration.Seconds())
}

func (m *Metrics) SetInFlight(endpoint string, count int) {
    m.inFlight[endpoint].Set(float64(count))
}
```

## 7. 优化检查清单

- [ ] 图片生成 < 100ms (P99 < 200ms)
- [ ] 验证响应 < 50ms (P99 < 100ms)
- [ ] 数据库查询 < 50ms
- [ ] Redis操作 < 5ms
- [ ] 启用图片缓存
- [ ] 使用连接池
- [ ] 实现限流
- [ ] 异步日志写入
- [ ] 预编译语句
- [ ] 监控P99延迟
- [ ] 启用Gzip压缩
- [ ] 静态资源缓存
