package realtime

import (
	"sync"

	"electronic-digital-signature/internal/app/usecase"
)

type Broker struct {
	mu          sync.RWMutex
	subscribers map[chan usecase.InboundPackageEvent]struct{}
}

func NewBroker() *Broker {
	return &Broker{
		subscribers: make(map[chan usecase.InboundPackageEvent]struct{}),
	}
}

func (b *Broker) Publish(event usecase.InboundPackageEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for subscriber := range b.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
}

func (b *Broker) Subscribe(buffer int) (<-chan usecase.InboundPackageEvent, func()) {
	if buffer <= 0 {
		buffer = 8
	}

	channel := make(chan usecase.InboundPackageEvent, buffer)

	b.mu.Lock()
	b.subscribers[channel] = struct{}{}
	b.mu.Unlock()

	return channel, func() {
		b.mu.Lock()
		if _, ok := b.subscribers[channel]; ok {
			delete(b.subscribers, channel)
			close(channel)
		}
		b.mu.Unlock()
	}
}
