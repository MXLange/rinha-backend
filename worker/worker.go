package worker

import (
	"fmt"
	"rinha-go/entities"
	"rinha-go/fetch"
	"rinha-go/repository"
	"sync"
)

type Worker struct {
	queue        chan *entities.Payment
	repo         *repository.Repository
	fetch        *fetch.Fetch
	donePayments map[string]bool
	mu           sync.Mutex // To protect donePayments map
}

func NewWorker(queue chan *entities.Payment, repo *repository.Repository, fetch *fetch.Fetch) (*Worker, error) {
	if queue == nil {
		return nil, fmt.Errorf("queue cannot be nil")
	}

	if repo == nil {
		return nil, fmt.Errorf("repository cannot be nil")
	}

	if fetch == nil {
		return nil, fmt.Errorf("fetch cannot be nil")
	}

	return &Worker{
		queue:        queue,
		repo:         repo,
		fetch:        fetch,
		donePayments: make(map[string]bool),
		mu:           sync.Mutex{},
	}, nil
}

func (w *Worker) Start(concurrency int) {
	for i := 0; i < concurrency; i++ {
		go func() {
			for payment := range w.queue {
				if payment == nil {
					continue
				}

				w.mu.Lock()
				if _, exists := w.donePayments[*payment.ID]; exists {
					w.mu.Unlock()
					continue // Skip already processed payments
				}
				w.mu.Unlock()

				err := w.processPayment(payment)
				if err != nil {
					fmt.Printf("Error processing payment: %v\n", err)
				}
			}
		}()
	}
}

func (w *Worker) processPayment(payment *entities.Payment) error {
	if payment == nil {
		return fmt.Errorf("payment cannot be nil")
	}

	// Simulate processing the payment
	fmt.Printf("Processing payment: %v\n", payment)

	p := payment.ToPaymentToSend()

	var err error
	var isDefault bool = payment.IsDefault

	if payment.ErrorEnum == "" {
		err, isDefault = w.fetch.SendPayment(p)
		if err != nil {
			w.queue <- payment
			return fmt.Errorf("failed to send payment: %v", err)
		}
		payment.IsDefault = isDefault
	}

	if isDefault {
		err = w.repo.SavePayment(p, true)
		if err != nil {
			payment.ErrorEnum = "SAVE"
			w.queue <- payment
			return fmt.Errorf("failed to save payment to default: %v", err)
		}
	} else {
		err = w.repo.SavePayment(p, false)
		if err != nil {
			payment.ErrorEnum = "SAVE"
			w.queue <- payment
			return fmt.Errorf("failed to save payment to fallback: %v", err)
		}
	}

	w.mu.Lock()
	w.donePayments[*payment.ID] = true
	w.mu.Unlock()

	return nil
}
