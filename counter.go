package ratelimite

import (
	"encoding/json"
	"time"
)

type CounterLimiter struct {
	baseLimiter
	// The time interval of each node
	PerIns time.Duration
	// current node
	CurrTime time.Time
	CurrNode *cicleNode `json:"-"`
	// for marshal
	Values []int64
	// sum of values
	Counts int64
}

func newCounter(bl *baseLimiter, ops ...Option) *CounterLimiter {
	for _, opt := range ops {
		opt(bl)
	}

	if bl.ins <= 0 {
		bl.ins = 1
	}

	l := &CounterLimiter{
		baseLimiter: *bl,
		PerIns:      bl.unit / time.Duration(bl.ins),
		CurrTime:    time.Now(),
		CurrNode:    newCicleLink(bl.ins),
		Counts:      0,
	}

	ato, err := l.atomicFunc(l.storageType, l.storageAddr)
	if err != nil {
		panic(err)
	}
	if l.storageType == REDIS_TYPE {
		ato.(*RedisAtomic).SetKey(l.name)
	}
	l.ato = ato

	return l
}

func (c *CounterLimiter) Release() (time.Duration, bool, error) {
	return c.ReleaseN(1)
}

func (c *CounterLimiter) ReleaseN(n int64) (time.Duration, bool, error) {
	if n > c.maxReq {
		return FOREVER, false, &LimiterError{"[ReleaseN] argument error"}
	}
	c.ato.Lock()
	defer c.ato.Unlock()

	if err := c.restore(); err != nil {
		return FOREVER, false, err
	}
	defer c.store()

	now := time.Now()
	span := now.Sub(c.CurrTime) / c.PerIns
	// 跨越间隔
	if span > 0 {
		c.CurrTime.Add(c.PerIns * span)
		// 移动表头，并清理移动过程中节点的值
		for i := 0; i < int(span); i++ {
			c.CurrNode = c.CurrNode.Next
			c.Counts -= c.CurrNode.Value
			c.CurrNode.Value = 0
		}
	}

	if c.Counts+n > c.maxReq {
		sp, cnt := 0, int64(0)
		node := c.CurrNode.Next
		for ; cnt < n; sp++ {
			cnt += node.Value
			node = node.Next
		}
		waitTime := time.Duration(sp) * c.PerIns
		return waitTime, false, nil
	}

	c.CurrNode.Value += n
	c.Counts += n
	c.CurrTime = now

	return 0, true, nil
}

func (c *CounterLimiter) Reset() error {
	c.ato.Lock()
	defer c.ato.Unlock()

	if err := c.restore(); err != nil {
		return err
	}
	defer c.store()

	c.Counts = 0
	c.CurrTime = time.Now()
	c.CurrNode.Value = 0

	node := c.CurrNode.Next
	for node != c.CurrNode {
		node.Value = 0
		node = node.Next
	}

	return nil
}

func (c *CounterLimiter) restore() error {
	data, err := c.ato.Restore()
	if err != nil || len(data) == 0 {
		return err
	}
	return c.deserializeFunc(data, c)
}

func (c *CounterLimiter) store() error {
	data, err := c.serializeFunc(c)
	if err != nil {
		return err
	}
	return c.ato.Store(data)
}

func CounterMarshal(v Limiter) ([]byte, error) {
	c := v.(*CounterLimiter)
	c.Values = c.CurrNode.Values()
	return json.Marshal(c)
}

func CounterUnmarshal(data []byte, v Limiter) error {
	c := v.(*CounterLimiter)
	if err := json.Unmarshal(data, c); err != nil {
		return err
	} else {
		c.CurrNode = newCicleLinkWithSlice(c.Values)
	}

	return nil
}

// 环形单链表，用于存储每个间隔的访问量
type cicleNode struct {
	Value int64
	Next  *cicleNode
}

func newCicleLink(len int) *cicleNode {
	s := make([]int64, len)
	return newCicleLinkWithSlice(s)
}

func newCicleLinkWithSlice(s []int64) *cicleNode {
	if len(s) == 0 {
		return nil
	}
	head := &cicleNode{Value: s[0]}
	next := head
	for _, v := range s[1:] {
		next.Next = &cicleNode{Value: v}
		next = next.Next
	}
	next.Next = head

	return head
}

func (c *cicleNode) Values() []int64 {
	vs := []int64{c.Value}
	for n := c.Next; n != c; n = n.Next {
		vs = append(vs, n.Value)
	}

	return vs
}
