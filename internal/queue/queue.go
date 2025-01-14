package queue

import (
	"context"

	"github.com/stjudewashere/seonaut/internal/httpcrawler"
)

type Queue struct {
	in     chan *httpcrawler.RequestMessage
	out    chan *httpcrawler.RequestMessage
	ack    chan string
	count  chan int
	active chan bool
}

func New(ctx context.Context) *Queue {
	q := Queue{
		in:     make(chan *httpcrawler.RequestMessage),
		out:    make(chan *httpcrawler.RequestMessage),
		ack:    make(chan string),
		count:  make(chan int),
		active: make(chan bool),
	}

	go q.manage(ctx)

	return &q
}

// Manage the queue with push, poll, and acknowledgement of elements in the queue.
func (q *Queue) manage(ctx context.Context) {
	defer func() {
		close(q.in)
		close(q.out)
		close(q.ack)
		close(q.count)
		close(q.active)
	}()

	queue := []*httpcrawler.RequestMessage{}
	active := make(map[string]bool)

	var first *httpcrawler.RequestMessage
	var out chan *httpcrawler.RequestMessage

	for {
		if first == nil && len(queue) > 0 {
			first = queue[0]
			active[first.URL] = true
			queue = queue[1:]
		}

		if first == nil {
			out = nil
		} else {
			out = q.out
		}

		select {
		case <-ctx.Done():
			return
		case q.count <- len(queue):
		case q.active <- (len(active) > 0 || len(queue) > 0):
		case v := <-q.in:
			queue = append(queue, v)
		case out <- first:
			first = nil
		case v := <-q.ack:
			delete(active, v)
		}
	}
}

// Adds a new value to the queue's end.
func (q *Queue) Push(value *httpcrawler.RequestMessage) {
	q.in <- value
}

// Returns the first element in the queue.
func (q *Queue) Poll() *httpcrawler.RequestMessage {
	return <-q.out
}

// Acknowledges a message has been processed.
func (q *Queue) Ack(s string) {
	q.ack <- s
}

// Returns the number of items currently in the queue.
func (q *Queue) Count() int {
	v, ok := <-q.count
	if !ok {
		return 0
	}

	return v
}

// Active returns true if the queue is not empty or has active elements.
func (q *Queue) Active() bool {
	return <-q.active
}
