package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/Prachi-Jamdade/shorten-url-fiber-redis/database"
	"github.com/Prachi-Jamdade/shorten-url-fiber-redis/helpers"
	"github.com/asaskevich/govalidator"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
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
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenURL(c *fiber.Ctx) error {
	body := new(request)

	if error := c.BodyParser(&body); error != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	clientIP := helpers.GetClientIP(c)

	// implement rate limiting
	redisClient2 := database.CreateClient(1)
	defer redisClient2.Close()

	val, err := redisClient2.Get(database.Context, clientIP).Result()
	if err == redis.Nil {
		_ = redisClient2.Set(database.Context, clientIP, os.Getenv("APP_QUOTA"), 30*60*time.Second).Err()
	} else {
		val, _ = redisClient2.Get(database.Context, clientIP).Result()
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := redisClient2.TTL(database.Context, clientIP).Result()
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Rate limit exceeded", "rate_limit_reset": limit / time.Nanosecond / time.Minute})
		}
	}

	// check if the input is an actual URL
	if !govalidator.IsURL(body.URL) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid URL"})
	}

	// check for domain error
	if !helpers.RemoveDomainError(body.URL) {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Service unavailable"})
	}

	// enforce https SSL
	// all url will be converted to https before storing in database
	body.URL = helpers.EnforceHTTP(body.URL)

	var id string

	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	redisClient := database.CreateClient(0)
	defer redisClient.Close()

	val, _ = redisClient.Get(database.Context, id).Result()

	if val != "" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "URL custom short is already in use"})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = redisClient.Set(database.Context, id, body.URL, body.Expiry*3600*time.Second).Err()

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to connect to server"})
	}

	resp := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}

	redisClient2.Decr(database.Context, clientIP)

	val, _ = redisClient2.Get(database.Context, clientIP).Result()
	resp.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := redisClient2.TTL(database.Context, clientIP).Result()
	resp.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	resp.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return c.Status(fiber.StatusOK).JSON(resp)
}
