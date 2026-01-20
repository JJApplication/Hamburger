/*
Project: Sandwich queue.go
Created: 2024/01/01 by Assistant
*/

package structure

import (
	"sync"
	"time"
)

// Queue 线程安全的队列结构
type Queue struct {
	items    []interface{} // 队列元素
	mutex    sync.RWMutex  // 读写锁
	capacity int           // 队列容量
	closed   bool          // 队列是否已关闭
	cond     *sync.Cond    // 条件变量，用于阻塞等待
}

// NewQueue 创建新的队列
func NewQueue(capacity int) *Queue {
	q := &Queue{
		items:    make([]interface{}, 0, capacity),
		capacity: capacity,
		closed:   false,
	}
	q.cond = sync.NewCond(&q.mutex)
	return q
}

// Enqueue 入队操作，如果队列满了则阻塞等待或丢弃
func (q *Queue) Enqueue(item interface{}) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// 如果队列已关闭，拒绝新元素
	if q.closed {
		return false
	}

	// 如果队列已满，直接丢弃（非阻塞策略）
	if len(q.items) >= q.capacity {
		return false
	}

	q.items = append(q.items, item)
	q.cond.Signal() // 通知等待的消费者
	return true
}

// EnqueueBlocking 阻塞式入队操作，如果队列满了则等待
func (q *Queue) EnqueueBlocking(item interface{}, timeout time.Duration) bool {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// 如果队列已关闭，拒绝新元素
	if q.closed {
		return false
	}

	// 等待队列有空间或超时
	deadline := time.Now().Add(timeout)
	for len(q.items) >= q.capacity && !q.closed {
		if timeout > 0 && time.Now().After(deadline) {
			return false // 超时
		}
		q.cond.Wait()
	}

	if q.closed {
		return false
	}

	q.items = append(q.items, item)
	q.cond.Signal() // 通知等待的消费者
	return true
}

// Dequeue 出队操作，如果队列为空则返回nil
func (q *Queue) Dequeue() interface{} {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if len(q.items) == 0 {
		return nil
	}

	item := q.items[0]
	q.items = q.items[1:]
	q.cond.Signal() // 通知等待的生产者
	return item
}

// DequeueBlocking 阻塞式出队操作，如果队列为空则等待
func (q *Queue) DequeueBlocking(timeout time.Duration) interface{} {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	deadline := time.Now().Add(timeout)
	for len(q.items) == 0 && !q.closed {
		if timeout > 0 && time.Now().After(deadline) {
			return nil // 超时
		}
		q.cond.Wait()
	}

	if len(q.items) == 0 {
		return nil // 队列已关闭且为空
	}

	item := q.items[0]
	q.items = q.items[1:]
	q.cond.Signal() // 通知等待的生产者
	return item
}

// Size 获取队列当前大小
func (q *Queue) Size() int {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.items)
}

// Capacity 获取队列容量
func (q *Queue) Capacity() int {
	return q.capacity
}

// IsEmpty 检查队列是否为空
func (q *Queue) IsEmpty() bool {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.items) == 0
}

// IsFull 检查队列是否已满
func (q *Queue) IsFull() bool {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return len(q.items) >= q.capacity
}

// Close 关闭队列，不再接受新元素
func (q *Queue) Close() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.closed = true
	q.cond.Broadcast() // 唤醒所有等待的goroutine
}

// IsClosed 检查队列是否已关闭
func (q *Queue) IsClosed() bool {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	return q.closed
}

// Clear 清空队列
func (q *Queue) Clear() {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	q.items = q.items[:0]
	q.cond.Broadcast()
}