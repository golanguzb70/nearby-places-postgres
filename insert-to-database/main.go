package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	totalRows  = 100_000_000 // Total number of rows to insert
	batchSize  = 3_000      // Rows per batch
	numWorkers = 4          // Number of concurrent insertion workers
)

type Place struct {
	Name        string
	Description string
	Latitude    float64
	Longitude   float64
}

// randomPlace returns one Place with random content
func randomPlace() Place {
	return Place{
		Name:        fmt.Sprintf("Place %d", rand.Intn(1_000_000_000)),
		Description: fmt.Sprintf("Description %d", rand.Intn(1_000_000_000)),
		Latitude:    -90 + rand.Float64()*180,  // range: -90 .. +90
		Longitude:   -180 + rand.Float64()*360, // range: -180 .. +180
	}
}

// insertPlacesBatch performs a single INSERT for all Places in 'places'.
// Uses a single VALUES list that has multiple tuples to do a bulk-like insert.
//   - E.g.: INSERT INTO places (...) VALUES ($1,$2,$3,$4),($5,$6,$7,$8) ...
func insertPlacesBatch(ctx context.Context, pool *pgxpool.Pool, places []Place) error {
	if len(places) == 0 {
		return nil
	}

	// Build INSERT
	sql := "INSERT INTO places (name, description, latitude, longitude) VALUES "
	args := make([]interface{}, 0, len(places)*4)
	argIndex := 1

	for i, place := range places {
		sql += fmt.Sprintf("($%d, $%d, $%d, $%d)", argIndex, argIndex+1, argIndex+2, argIndex+3)
		if i < len(places)-1 {
			sql += ","
		}

		args = append(args, place.Name, place.Description, place.Latitude, place.Longitude)
		argIndex += 4
	}

	_, err := pool.Exec(ctx, sql, args...)
	return err
}

func main() {
	// OPTIONAL: Seeding the random generator
	// rand.Seed(time.Now().UnixNano())

	// Read port from env, fallback to 5433
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5433"
	}

	// Build Postgres connection string
	dbURL := fmt.Sprintf("postgres://nearby-places:12345@localhost:%s/nearby-places", dbPort)

	// Create a connection pool
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pool.Close()

	log.Printf("Starting insertion of %d rows in batches of %d with %d workers...", totalRows, batchSize, numWorkers)

	// We'll create a channel of []Place to pass batches from the producer to workers
	batches := make(chan []Place, numWorkers) // channel buffer = numWorkers or larger

	// 1) Start the worker pool
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for w := 1; w <= numWorkers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for chunk := range batches {
				if err := insertPlacesBatch(ctx, pool, chunk); err != nil {
					// Log the error. For real usage, you might want to handle it carefully,
					// possibly re-queue, or record failed chunk, etc.
					log.Printf("[Worker %d] Insert failed: %v", workerID, err)
					return
				}
				// You can control how chatty logging is if 20 million rows is large
				log.Printf("[Worker %d] Inserted a chunk of %d rows", workerID, len(chunk))
			}
		}(w)
	}

	// 2) Producer: generate random data in batches and send them to the channel
	go func() {
		defer close(batches) // close channel when all batches have been sent

		// number of full batches
		numFullBatches := totalRows / batchSize
		remainder := totalRows % batchSize

		for i := 0; i < numFullBatches; i++ {
			chunk := make([]Place, 0, batchSize)
			for j := 0; j < batchSize; j++ {
				chunk = append(chunk, randomPlace())
			}
			batches <- chunk
		}

		// remainder, if any
		if remainder > 0 {
			chunk := make([]Place, 0, remainder)
			for j := 0; j < remainder; j++ {
				chunk = append(chunk, randomPlace())
			}
			batches <- chunk
		}
	}()

	// 3) Wait for workers to finish
	wg.Wait()

	log.Println("All rows inserted successfully.")
}
