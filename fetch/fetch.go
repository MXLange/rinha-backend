package fetch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"rinha-go/entities"
	"sync"
	"time"

	"github.com/MXLange/royalfetch/v2"
)

type Fetch struct {
	principal      *royalfetch.RoyalFetch
	fallback       *royalfetch.RoyalFetch
	sendToFallback bool
	wait           bool
	isToClose      bool
	mu             sync.Mutex
}

func NewFetch(baseDefault, baseFallback string) (*Fetch, error) {

	if baseDefault == "" || baseFallback == "" {
		return nil, fmt.Errorf("base URLs cannot be empty")
	}

	p := royalfetch.New(royalfetch.RoyalFetch{
		BaseURL: baseDefault,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, &http.Client{
		Timeout: 0,
	})

	f := royalfetch.New(royalfetch.RoyalFetch{
		BaseURL: baseFallback,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, &http.Client{
		Timeout: 0,
	})

	return &Fetch{
		principal: p,
		fallback:  f,
		mu:        sync.Mutex{},
	}, nil
}

type Health struct {
	failing         bool `json:"failing"`
	minResponseTime int  `json:"minResponseTime"`
}

func (f *Fetch) CheckStatus() {
	go func() {
		ticker := time.NewTicker(5200 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {

			if f.isToClose {
				break
			}

			hDefault := &Health{}
			hFallback := &Health{}

			wg := sync.WaitGroup{}
			wg.Add(2)

			go f.GetHealthDefault(&wg, hDefault)
			go f.GetHealthFallback(&wg, hFallback)

			wg.Wait()

			f.setIsToSendToFallback(hDefault, hFallback)
		}

	}()
}

func (f *Fetch) Done() {
	f.isToClose = true
}

func (f *Fetch) setIsToSendToFallback(hd, hf *Health) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if hd == nil && hf == nil {
		f.sendToFallback = false
		f.wait = false
		return
	}

	if hd == nil || hf == nil {
		f.sendToFallback = false
		return
	}
	if hd.failing && hf.failing {
		f.wait = true
		f.sendToFallback = false
		return
	}

	if !hd.failing {
		f.sendToFallback = false
		f.wait = false

		proportion := float64(hd.minResponseTime) / float64(hf.minResponseTime)
		if proportion >= 2.0 {
			f.sendToFallback = true
			f.wait = false
			return
		}

		return
	}

	if hd.failing && !hf.failing {
		f.sendToFallback = true
		f.wait = false
		return
	}

}

func (f *Fetch) SendPayment(p *entities.PaymentToSend) (error, bool) {
	if p == nil {
		return fmt.Errorf("payment cannot be nil"), false
	}

	f.mu.Lock()
	sendToFallback := f.sendToFallback
	f.mu.Unlock()

	for f.wait {
		time.Sleep(1000 * time.Millisecond)
	}

	if sendToFallback {
		return f.sendPaymentFallback(p), false
	}
	return f.sendPaymentDefault(p), true
}

func (f *Fetch) sendPaymentDefault(p *entities.PaymentToSend) error {
	body, err := json.Marshal(p)
	fmt.Println("Sending payment to principal service:", string(body))
	if err != nil {
		return err
	}

	res, err := f.principal.Post("/payments", string(body))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send payment, status code: %d", res.StatusCode)
	}

	return nil
}

func (f *Fetch) sendPaymentFallback(p *entities.PaymentToSend) error {
	body, err := json.Marshal(p)
	fmt.Println("Sending payment to principal service:", string(body))
	if err != nil {
		return err
	}

	res, err := f.fallback.Post("/payments", string(body))
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send payment to fallback, status code: %d", res.StatusCode)
	}

	return nil
}

func (f *Fetch) GetHealthDefault(wg *sync.WaitGroup, h *Health) {
	defer wg.Done()
	res, err := f.principal.Get("/payments/service-health")
	if err != nil {
		h = nil
		return
	}

	if res.StatusCode != http.StatusOK {
		h = nil
		return
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		h = nil
		return
	}

	health := &Health{}
	if err := json.Unmarshal(resBody, health); err != nil {
		h = nil
		return
	}

	h = health
}

func (f *Fetch) GetHealthFallback(wg *sync.WaitGroup, h *Health) {
	defer wg.Done()
	res, err := f.fallback.Get("/payments/service-health")
	if err != nil {
		h = nil
		return
	}

	if res.StatusCode != http.StatusOK {
		h = nil
		return
	}

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		h = nil
		return
	}

	health := &Health{}
	if err := json.Unmarshal(resBody, health); err != nil {
		h = nil
		return
	}

	h = health
}
