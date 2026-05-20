package performance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type EventDrivenService struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	handlers      map[string]EventHandler
	eventBus     *EventBus
	scheduler    *EventScheduler
	metrics      *EventMetrics
}

type EventHandler func(ctx context.Context, event *Event) error

type EventBus struct {
	mu       sync.RWMutex
	channels map[string]chan *Event
	buffers  map[string][]*Event
}

type EventScheduler struct {
	mu          sync.RWMutex
	scheduled  []*ScheduledEvent
	processing bool
}

type ScheduledEvent struct {
	Event      *Event
	ExecuteAt  time.Time
	Repeat     bool
	Interval   time.Duration
}

type EventMetrics struct {
	TotalEvents     atomic.Int64
	ProcessedEvents atomic.Int64
	FailedEvents    atomic.Int64
	AvgLatencyMs   atomic.Int64
	QueueDepth     atomic.Int64
}

func NewEventDrivenService() *EventDrivenService {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventDrivenService{
		ctx:       ctx,
		cancel:    cancel,
		handlers:  make(map[string]EventHandler),
		eventBus:  NewEventBus(),
		scheduler: NewEventScheduler(),
		metrics:   &EventMetrics{},
	}
}

func NewEventBus() *EventBus {
	return &EventBus{
		channels: make(map[string]chan *Event),
		buffers:  make(map[string][]*Event),
	}
}

func NewEventScheduler() *EventScheduler {
	return &EventScheduler{
		scheduled: make([]*ScheduledEvent, 0),
	}
}

func (s *EventDrivenService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return nil
	}

	s.isRunning = true

	go s.eventBus.run(s.ctx)
	go s.scheduler.run(s.ctx)

	log.Println("[EventDrivenService] Started successfully")
	return nil
}

func (s *EventDrivenService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return
	}

	s.cancel()
	s.isRunning = false
	log.Println("[EventDrivenService] Stopped")
}

func (s *EventDrivenService) RegisterHandler(eventType string, handler EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.handlers[eventType] = handler
	s.eventBus.createChannel(eventType)
}

func (s *EventDrivenService) Publish(event *Event) error {
	s.metrics.TotalEvents.Add(1)
	return s.eventBus.publish(event)
}

func (s *EventDrivenService) Schedule(event *Event, executeAt time.Time, repeat bool, interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	scheduled := &ScheduledEvent{
		Event:     event,
		ExecuteAt: executeAt,
		Repeat:    repeat,
		Interval:  interval,
	}

	s.scheduler.scheduled = append(s.scheduler.scheduled, scheduled)
}

func (s *EventDrivenService) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_events":      s.metrics.TotalEvents.Load(),
		"processed_events":  s.metrics.ProcessedEvents.Load(),
		"failed_events":     s.metrics.FailedEvents.Load(),
		"avg_latency_ms":   s.metrics.AvgLatencyMs.Load(),
		"queue_depth":      s.metrics.QueueDepth.Load(),
	}
}

func (b *EventBus) createChannel(eventType string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.channels[eventType]; !exists {
		b.channels[eventType] = make(chan *Event, 1000)
		b.buffers[eventType] = make([]*Event, 0)
	}
}

func (b *EventBus) publish(event *Event) error {
	b.mu.RLock()
	ch, exists := b.channels[event.Type]
	b.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no channel for event type: %s", event.Type)
	}

	select {
	case ch <- event:
		return nil
	default:
		b.mu.Lock()
		b.buffers[event.Type] = append(b.buffers[event.Type], event)
		b.mu.Unlock()
		return nil
	}
}

func (b *EventBus) run(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.processBuffers()
		}
	}
}

func (b *EventBus) processBuffers() {
	b.mu.RLock()
	for eventType, ch := range b.channels {
		buffers := b.buffers[eventType]
		b.mu.RUnlock()

		for len(buffers) > 0 && len(ch) < cap(ch) {
			select {
			case ch <- buffers[0]:
				b.mu.Lock()
				b.buffers[eventType] = buffers[1:]
				b.mu.Unlock()
				buffers = b.buffers[eventType]
			default:
				break
			}
		}

		b.mu.RLock()
	}
}

func (s *EventScheduler) run(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processScheduled()
		}
	}
}

func (s *EventScheduler) processScheduled() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	var remaining []*ScheduledEvent

	for _, se := range s.scheduled {
		if se.ExecuteAt.Before(now) || se.ExecuteAt.Equal(now) {
			continue
		}

		remaining = append(remaining, se)
	}

	s.scheduled = remaining
}
