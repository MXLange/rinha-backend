package handlers

import (
	"fmt"
	"rinha-go/entities"
	"rinha-go/repository"

	"github.com/gofiber/fiber/v2"
)

type Handlers struct {
	queue chan *entities.Payment
	repo  *repository.Repository // Assuming you have a repository package
}

func NewHandlers(queue chan *entities.Payment, repo *repository.Repository) (*Handlers, error) {

	if queue == nil {
		return nil, fmt.Errorf("queue cannot be nil")
	}

	if repo == nil {
		return nil, fmt.Errorf("repository cannot be nil")
	}

	return &Handlers{
		queue: queue,
		repo:  repo,
	}, nil
}

func (h *Handlers) Publish(c *fiber.Ctx) error {
	var payment *entities.Payment = &entities.Payment{}
	if err := c.BodyParser(payment); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if payment.ID == nil || payment.Amount == nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing required fields: correlationId and amount",
		})
	}

	h.queue <- payment

	return c.SendStatus(fiber.StatusCreated)
}

func (h *Handlers) GetSummary(c *fiber.Ctx) error {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	var f, t *string = nil, nil
	if fromStr != "" {
		f = &fromStr
	}
	if toStr != "" {
		t = &toStr
	}
	// Chama o método para buscar o summary no Redis
	summary, err := h.repo.GetPaymentsSummary(f, t)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get summary",
		})
	}

	return c.JSON(summary)
}
