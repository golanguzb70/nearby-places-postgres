package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Place defines the data model for a place
type Place struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Distance    float64   `json:"distance_km"`
}

var pool *pgxpool.Pool

// getPlaces handles GET /places and returns a list of places
func getPlaces(c *gin.Context) {
	ctx := context.Background()
	lat := c.Query("lat")
	lon := c.Query("lon")
	radius := c.Query("radius")

	if lat == "" || lon == "" || radius == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat, lon, and radius query parameters are required"})
		return
	}

	page := c.DefaultQuery("page", "1")
	limit := c.DefaultQuery("limit", "10")
	pageInt, _ := strconv.Atoi(page)
	limitInt, _ := strconv.Atoi(limit)

	// Query the database for places within the specified radius
	rows, err := pool.Query(ctx, `
        SELECT 
			id, name, description, latitude, longitude, created_at, updated_at, 
			6371 * ACOS(
				COS(RADIANS($1))
			* COS(RADIANS(p.latitude))
			* COS(RADIANS(p.longitude) - RADIANS($2))
			+ SIN(RADIANS($1))
			* SIN(RADIANS(p.latitude))
			) AS distance_km
		FROM places p
		WHERE
		p.latitude BETWEEN ($1 - ($3 / 111.0)) AND ($1 + ($3 / 111.0))
		AND p.longitude BETWEEN ($2 - ($3 / (111.0 * COS(RADIANS($1))))) AND ($2 + ($3 / (111.0 * COS(RADIANS($1)))))
		AND 6371 * ACOS(
				COS(RADIANS($1))
			* COS(RADIANS(p.latitude))
			* COS(RADIANS(p.longitude) - RADIANS($2))
			+ SIN(RADIANS($1))
			* SIN(RADIANS(p.latitude))
			) <= $3
		LIMIT $4 OFFSET $5;
		`, lat, lon, radius, limitInt, (pageInt-1)*limitInt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	places := []Place{}
	for rows.Next() {
		var p Place
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Latitude, &p.Longitude, &p.CreatedAt, &p.UpdatedAt, &p.Distance); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		places = append(places, p)
	}

	c.JSON(http.StatusOK, places)
}

func main() {
	// Build the connection string. You can override via env var DATABASE_URL.
	dbURL := fmt.Sprintf("postgres://nearby-places:12345@localhost:5433/nearby-places")

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		log.Fatalf("Unable to parse config: %v", err)
	}

	// Set the maximum number of connections in the pool
	config.MaxConns = 100 // adjust this value as needed

	// Create a new pgx pool connection
	ctx := context.Background()
	pool, err = pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Unable to create pool: %v", err)
	}
	defer pool.Close()

	// Set up Gin router
	router := gin.Default()
	router.GET("/places", getPlaces)

	// Run server on port from env variable PORT, default to 8080
	port := "9090"

	log.Printf("Starting server on port %s...", port)
	router.Run(":" + port)
}
