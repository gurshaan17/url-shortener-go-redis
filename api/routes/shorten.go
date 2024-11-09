package routes

import (
	"time"
	"os"
	"strconv"

	"github.com/gurshaan17/url-shortener-go-redis/database"
	"github.com/gurshaan17/url-shortener-go-redis/helpers"
	"github.com/gofiber/fiber/v2"
	"github.com/go-redis/redis/v8"
	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateLimiting   int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)
	if err := c.BodyParser(&body); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "can't parse json"})
	}

	// Implement rate limit & check if input is an actual URL
	r2 := database.CreateClient(1)
	defer r2.Close()

	// Get the current rate limit value for the IP
	val, err := r2.Get(database.Ctx, c.IP()).Result()
	var valInt int
	if err == redis.Nil {
		// If no value exists, set the initial rate limit
		initialQuota := os.Getenv("API_QUOTA")
		if initialQuota == "" {
			initialQuota = "10" // Default value if not set
		}
		valInt, _ = strconv.Atoi(initialQuota)
		_ = r2.Set(database.Ctx, c.IP(), valInt, 30*time.Minute).Err()
	} else if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "cannot connect to database"})
	} else {
		valInt, _ = strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(429).JSON(fiber.Map{
				"error":            "rate limit exceeded",
				"rate_limit_reset": limit / time.Minute,
			})
		}
	}

	if !govalidator.IsURL(body.URL) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid URL"})
	}

	// Check for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(503).JSON(fiber.Map{"error": "you can't access this URL"})
	}

	// Enforce HTTPS, SSL
	body.URL = helpers.EnforceHTTP(body.URL)

	// Generate ID
	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()

	// Check if the custom short ID is already in use
	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(403).JSON(fiber.Map{
			"error": "Custom short URL is already in use",
		})
	}

	// Set expiry time to default 24 hours if not provided
	if body.Expiry == 0 {
		body.Expiry = 24
	}

	// Set the URL in the Redis database with the specified expiry
	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Unable to connect to server",
		})
	}

	// Prepare the response
	resp := response{
		URL:             body.URL,
		CustomShort:     os.Getenv("DOMAIN") + "/" + id,
		Expiry:          body.Expiry,
		XRateLimiting:   valInt,
		XRateLimitReset: 30,
	}

	// Decrement the rate limit after processing the request
	r2.Decr(database.Ctx, c.IP())

	// Update the response with the current rate limit and reset time
	val, _ = r2.Get(database.Ctx, c.IP()).Result()
	resp.XRateLimiting, _ = strconv.Atoi(val)
	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Minute

	return c.Status(200).JSON(resp)
}