// services/order-service/internal/worker/worker.go

package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// periodicWorker پایه‌ای‌ترین واحد مشترک بین تمام ورکرهای این
// پکیج است. مدیریت ticker، graceful shutdown، و جلوگیری از
// هم‌پوشانی tickها را بر عهده دارد.
//
// هر ورکر یک tickFn تزریق می‌کند که منطق اختصاصی آن ورکر را
// اجرا می‌کند. این الگو تکرار کدهای boilerplate را حذف می‌کند
// در حالی که هر ورکر مسئولیت تکی خود را دارد
type periodicWorker struct {
	name     string
	interval time.Duration
	tickFn   func(ctx context.Context) error

	running  atomic.Bool
	wg       sync.WaitGroup
	stopChan chan struct{}
}

func newPeriodicWorker(
	name string,
	interval time.Duration,
	tickFn func(ctx context.Context) error,
) *periodicWorker {
	return &periodicWorker{
		name:     name,
		interval: interval,
		tickFn:   tickFn,
		stopChan: make(chan struct{}),
	}
}

// start گوروتیین ورکر را اجرا کرده و آن را در WaitGroup ثبت می‌کند
func (w *periodicWorker) start() {
	w.wg.Add(1)
	go w.run()
}

// stop ورکر را متوقف کرده و تا زمان اتمام آخرین tick جاری
// یا پایان مهلت stopCtx منتظر می‌ماند
func (w *periodicWorker) stop(stopCtx context.Context) error {
	close(w.stopChan)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("order-service: %s stopped gracefully", w.name)
		return nil
	case <-stopCtx.Done():
		log.Printf("order-service: %s stop timed out", w.name)
		return stopCtx.Err()
	}
}

func (w *periodicWorker) run() {
	defer w.wg.Done()

	log.Printf(
		"order-service: %s started (interval=%s)",
		w.name,
		w.interval,
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// اجرای اولیه بلافاصله پس از استارت
	w.tick()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.tick()
		}
	}
}

func (w *periodicWorker) tick() {

	// اگر دور قبلی هنوز تمام نشده، این دور را رد کن
	if !w.running.CompareAndSwap(false, true) {
		log.Printf(
			"order-service: %s previous tick still in progress, skipping",
			w.name,
		)
		return
	}
	defer w.running.Store(false)

	tickCtx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer cancel()

	if err := w.tickFn(tickCtx); err != nil {
		log.Printf(
			"order-service: %s tick failed: %v",
			w.name,
			err,
		)
	}
}
