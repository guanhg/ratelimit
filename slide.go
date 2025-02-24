package ratelimite

import (
	"sync"
	"sync/atomic"
)

// 滑动窗口统计
type Bucket struct {
	StartMs int64 // 节点启动时间 以毫秒ms为单位
	Val     int64 // 计数
}

func (b *Bucket) Add(v int64) {
	atomic.AddInt64(&b.Val, v)
}

func (b *Bucket) Get() int64 {
	return atomic.LoadInt64(&b.Val)
}

func (b *Bucket) Reset(t int64) {
	atomic.StoreInt64(&b.StartMs, t)
	atomic.StoreInt64(&b.Val, 0)
}

type SlideBucket struct {
	BucketNum  int
	BucketInMs int64
	Mux        sync.Mutex // 数组更新需要加锁
	List       []*Bucket
}

func NewSolideBucket(num int, ms int64, now int64) *SlideBucket {
	w := new(SlideBucket)
	w.BucketNum = num
	w.BucketInMs = ms
	w.List = make([]*Bucket, num)
	// 循环数组初始化
	idx := w.IndexOfTime(now)
	startMs := w.caclStartTime(now)
	for i := idx; i < num; i++ {
		b := &Bucket{Val: 0, StartMs: startMs}
		w.List[i] = b
		startMs += w.BucketInMs
	}
	for i := 0; i < idx; i++ {
		b := &Bucket{Val: 0, StartMs: startMs}
		w.List[i] = b
		startMs += w.BucketInMs
	}
	return w
}

// t是StatBucket启动后经历的时间，StatBucket启动时间默认为0毫秒
func (s *SlideBucket) IndexOfTime(t int64) int {
	return int(t/s.BucketInMs) % s.BucketNum
}

func (s *SlideBucket) caclStartTime(t int64) int64 {
	return t - (t % s.BucketInMs)
}

// 获取当前时间该更新的bucket，需要考虑并发安全问题
func (s *SlideBucket) CurrBucketOfTime(t int64) *Bucket {
	idx := s.IndexOfTime(t)
	start := s.caclStartTime(t)
	bucket := s.List[idx]

	if start == atomic.LoadInt64(&bucket.StartMs) { // 刚好取得当前bucket
		return bucket
	} else if start > atomic.LoadInt64(&bucket.StartMs) { // s.List数组已经经历了一个循环，当前bucket已经过期，需要重置
		bucket.Reset(start)
		return bucket
	} else if start < atomic.LoadInt64(&bucket.StartMs) {
		if s.BucketNum == 1 { // 窗口只有一个bucket，在高并发下，有可能发生
			return bucket
		}
	}

	return nil
}

func (s *SlideBucket) AddWithTime(v int, t int64) {
	bucket := s.CurrBucketOfTime(t)
	bucket.Add(int64(v))
}

func (s *SlideBucket) isBucketExpired(b *Bucket, t int64) bool {
	bt := atomic.LoadInt64(&b.StartMs)
	return t-bt > s.BucketInMs*int64(s.BucketNum)
}

func (s *SlideBucket) ValuesWithTime(t int64) []*Bucket {
	values := make([]*Bucket, 0)
	for _, b := range s.List {
		if b == nil || s.isBucketExpired(b, t) {
			continue
		}
		values = append(values, b)
	}
	return values
}

func (s *SlideBucket) Count(t int64) int {
	var count int64
	for _, b := range s.ValuesWithTime(t) {
		count += b.Get()
	}

	return int(count)
}
