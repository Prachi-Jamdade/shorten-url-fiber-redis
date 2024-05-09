package routes

import (
	"github.com/Prachi-Jamdade/shorten-url-fiber-redis/database"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

func ResolveURL(c *fiber.Ctx) error {
	url := c.Params("url")

	redisClient := database.CreateClient(0)
	defer redisClient.Close()

	value, err := redisClient.Get(database.Context, url).Result()

	if err == redis.Nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "URL not found in database"})
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Cannot connect to database"})
	}

	redisClientInr := database.CreateClient(1)
	defer redisClientInr.Close()

	_ = redisClientInr.Incr(database.Context, "counter")

	// redirect to original URL
	return c.Redirect(value, 301)

}
