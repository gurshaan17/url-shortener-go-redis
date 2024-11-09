package routes

import (
	"time"
	"github.com/gurshaan17/url-shortener-go-redis/database"
	"github.com/gurshaan17/url-shortener-go-redis/helpers"
	"github.com/gofiber/fiber/v2"
	"os"
	"strconv"
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

	//implement rate limit & check if input is an actual URL
	r2 := database.CreateClient(1)
	defer r2.Close()
	val, err := r2.Get(database.Ctx, c.IP()).Result()
	if err == redis.Nil{ 
		_ = r2.Set(database.Ctx, c.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ := r2.Get(database.Ctx, c.IP()).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, c.IP()).Result()
			return c.Status(500).JSON(fiber.Map{
				"error": "rate limit exceeded",
				"rate_limit_reset": limit / time.Nanosecond / time.Minute,
			})
		}
	}


	if !govalidator.IsURL(body.URL) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid URL"})
	}


	// check for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(503).JSON(fiber.Map{"error": "you can't access it"})
	}
	// enforce https, ssl
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()
	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
		return c.Status(403).JSON(fiber.Map{
			"error": "URL Custom short is already in use",
		})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Unable to connect to server",
		})
	}

	resp := response{
		URL: 				body.URL,
		CustomShort:        "",
		Expiry:  			body.Expiry,
		XRateRemaining: 	10,
		XRateLimitReset: 	30,
	}

	r2.Decr(database.Ctx, c.IP())

	val, _ := r2.Get(database.Ctx, c.IP()).Result()
	resp.RateRemaining, _ := strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, c.IP()).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute
	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return c.Status(200).JSON(resp)
}
