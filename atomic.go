package ratelimite

import (
	"sync"
)

func FetchStorageAtomic(tp StorageType, addr string) (Atomic, error) {
	switch tp {
	case PROCESS_TYPE:
		return &ProcessAtomic{}, nil
	case REDIS_TYPE:
		return NewRedisAtomic(addr)
	default:
		panic(LimiterError{"StorageType missing"})
	}
}

type Atomic interface {
	sync.Locker

	Store([]byte) error
	Restore() ([]byte, error)
}

type ProcessAtomic struct {
	sync.Mutex
}

func (p *ProcessAtomic) Store([]byte) error {
	return nil
}

func (p *ProcessAtomic) Restore() ([]byte, error) {
	return nil, nil
}
