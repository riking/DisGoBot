package discourse

import "time"
import log "github.com/riking/DisGoBot/logging"

type DRateLimiter struct {
	slots        []time.Time
	eventExpiry  time.Duration
	ch           <-chan struct{}
	close        func()
}

func NewDRateLimiter(max int, expiry time.Duration) *DRateLimiter {
	rl := new(DRateLimiter)
	rl.slots = make([]time.Time, 0, max)
	rl.eventExpiry = expiry
	rateChan := make(chan struct{})
	rl.ch = rateChan
	closeChan := make(chan struct{})
	rl.close = func() {
		closeChan <- struct{}{}
	}
	go rl.emit(rateChan, closeChan)
	return rl
}

func (rl DRateLimiter) DoC(f func()) {
	<-rl.ch
	f()
}

func (rl DRateLimiter) Wait() {
	<-rl.ch
}

func (rl DRateLimiter) Timeout(d time.Duration, f func()) bool {
	select {
	case <-rl.ch:
	case <-time.After(d):
		return false
	}
	f()
	return true
}

func (rl DRateLimiter) Close() {
	rl.close()
}

// ---

func (rl DRateLimiter) emit(channel chan <- struct{}, closeChan <-chan struct{}) {
	for {
		select {
		case <-closeChan:
			return
		case channel <- struct{}{}:
			rl.performed()
		}
	}
}

func (rl DRateLimiter) performed() {
	if len(rl.slots) > 0 {
		newest := rl.slots[0]
		oldest := rl.slots[len(rl.slots) - 1]
		if newest.Add(time.Duration(float64(rl.eventExpiry) * 2.02)).Before(time.Now()) {
			// Ratelimit structure in Redis has expired. (2% margin)
			// Dump the timestamps
			rl.slots = make([]time.Time, 0, cap(rl.slots))
		} else if oldest.Add(rl.eventExpiry).Before(time.Now()) {
			// Oldest has expired, we have a use
		} else {
			// Have to wait.
			sleepTime := time.Now().Sub(oldest.Add(rl.eventExpiry))
			log.Warn("Ratelimit triggered! Waiting for", sleepTime)
			time.Sleep(sleepTime)
		}
	}

	// Put now on the end
	rl.slots = append(rl.slots, time.Now())
	if len(rl.slots) == cap(rl.slots) {
		// Trim if full
		rl.slots = rl.slots[1:len(rl.slots)]
	}
}
