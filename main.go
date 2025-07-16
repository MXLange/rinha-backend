package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"rinha-go/entities"
	"rinha-go/fetch"
	"rinha-go/handlers"
	"rinha-go/repository"
	"rinha-go/worker"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {

	app := fiber.New(fiber.Config{
		AppName: "My Rinha API",
	})
	defer app.Shutdown()

	c := make(chan *entities.Payment, 10000)

	connStr := fmt.Sprintf(
		"postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		os.Getenv("POSTGRES_USER"),     // usuário
		os.Getenv("POSTGRES_PASSWORD"), // senha
		os.Getenv("POSTGRES_HOST"),     // host
		"5432",                         // porta
		os.Getenv("POSTGRES_DB"),       // banco de dados
	)

	fmt.Println("Connecting to database with connection string:", connStr)
	// Conectar
	conn, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		log.Fatal("Erro ao conectar:", err)
	}
	defer conn.Close()

	conn.Config().MaxConnIdleTime = 5 * time.Minute
	conn.Config().MaxConnLifetime = 30 * time.Minute
	conn.Config().MaxConns = 450
	conn.Config().MinConns = 50
	conn.Config().HealthCheckPeriod = 30 * time.Second

	r, err := repository.NewRepository(conn)
	if err != nil {
		fmt.Printf("Error creating repository: %v\n", err)
		panic(err)
	}

	err = r.CreateTables()
	if err != nil {
		fmt.Printf("Error creating tables: %v\n", err)
		panic(err)
	}

	h, err := handlers.NewHandlers(c, r)
	if err != nil {
		fmt.Printf("Error creating handlers: %v\n", err)
		panic(err)
	}

	f, err := fetch.NewFetch(os.Getenv("BASE_DEFAULT_URL"), os.Getenv("BASE_FALLBACK_URL"))
	if err != nil {
		fmt.Printf("Error creating fetch: %v\n", err)
		panic(err)
	}

	w, err := worker.NewWorker(c, r, f)
	if err != nil {
		fmt.Printf("Error creating worker: %v\n", err)
		panic(err)
	}

	nStr := os.Getenv("WORKER_CONCURRENCY")

	n, err := strconv.Atoi(nStr)
	if err != nil {
		fmt.Printf("Error converting WORKER_CONCURRENCY to int: %v\n", err)
		panic(err)
	}

	w.Start(n)

	app.Post("/payments", h.Publish)
	app.Get("/payments-summary", h.GetSummary)

	fmt.Printf("Running with concurrency: %d\n", n)

	err = app.Listen(":8080")
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		panic(err)
	}
}
