package ratelimite

import (
	"sync"
)

type AtomicType uint

const (
	PROCESS AtomicType = 0
	REDIS   AtomicType = 1
)

type locker interface {
	Lock() error
	Unlock() (bool, error)
}

type Atomic interface {
	locker
	storage
	getType() AtomicType
}

func NewAtomic(at AtomicType) Atomic {
	switch at {
	case REDIS:
		return &RedisAtomic{}
	case PROCESS:
		return &ProcessAtomic{}
	default:
		return nil
	}
}

type ProcessAtomic struct {
	ProcessLocker
	defaultStorage
}

type ProcessLocker struct {
	mux sync.Mutex
}

func (pl *ProcessLocker) Lock() error {
	pl.mux.Lock()
	return nil
}

func (pl *ProcessLocker) Unlock() (bool, error) {
	pl.mux.Unlock()
	return true, nil
}

func (pl *ProcessAtomic) getType() AtomicType {
	return PROCESS
}

type storage interface {
	Store([]byte) error
	Restore() ([]byte, error)
}

type defaultStorage struct{}

func (p *defaultStorage) Store([]byte) error {
	return nil
}

func (p *defaultStorage) Restore() ([]byte, error) {
	return nil, nil
}
