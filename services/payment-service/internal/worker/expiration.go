// services/payment-service/internal/worker/expiration.go

package worker

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// StalePaymentExpirer اینترفیس اختصاصی ورکر جهت جداسازی وابستگی
// از کل PaymentService
type StalePaymentExpirer interface {
	ExpireStalePayments(ctx context.Context, timeout time.Duration) error
}

// ExpirationWorker مدیریت اجرای دوره‌ای منقضی کردن پرداخت‌های بلاتکلیف
// را بر عهده دارد
type ExpirationWorker struct {
	service      StalePaymentExpirer
	interval     time.Duration
	staleTimeout time.Duration
	running      atomic.Bool    // برای جلوگیری از هم‌پوشانی اجراها
	wg           sync.WaitGroup // برای تضمین Graceful Shutdown
	stopChan     chan struct{}  // سیگنال توقف به goroutine اصلی
}

// Start گوروتیین ورکر را اجرا کرده و آن را در WaitGroup ثبت می‌کند
func (w *ExpirationWorker) Start() {
	w.wg.Add(1)
	go w.run()
}

// Stop ورکر را متوقف کرده و تا زمان اتمام
// آخرین tick جاری یا پایان مهلت stopCtx منتظر می‌ماند
func (w *ExpirationWorker) Stop(stopCtx context.Context) error {
	close(w.stopChan)

	done := make(chan struct{})
	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("payment-service: expiration worker stopped gracefully")
		return nil
	case <-stopCtx.Done():
		log.Println("payment-service: expiration worker stop timed out")
		return stopCtx.Err()
	}
}

func (w *ExpirationWorker) run() {
	defer w.wg.Done()

	log.Printf(
		"payment-service: expiration worker started (interval=%s, stale_timeout=%s)",
		w.interval,
		w.staleTimeout,
	)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// اجرای اولیه در زمان استارت
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

// tick یک دور اجرای انقضا را انجام می‌دهد. برای جلوگیری از
// بلاک شدن ticker در صورت طولانی شدن یک دور، از یک timeout
// اختصاصی استفاده می‌کنیم
func (w *ExpirationWorker) tick() {
	// اگر دور قبلی هنوز تمام نشده، این دور را رد کن تا هم‌پوشانی رخ ندهد
	if !w.running.CompareAndSwap(false, true) {
		log.Println(
			"payment-service: previous expiration tick still in progress, skipping",
		)
		return
	}
	defer w.running.Store(false)

	// ساخت context اختصاصی برای این tick جهت جلوگیری از معلق ماندن کوئری‌ها
	tickCtx, cancel := context.WithTimeout(
		context.Background(),
		2*time.Minute,
	)
	defer cancel()

	if err := w.service.ExpireStalePayments(
		tickCtx,
		w.staleTimeout,
	); err != nil {

		log.Printf(
			"payment-service: expiration worker tick failed: %v",
			err,
		)
	}
}
