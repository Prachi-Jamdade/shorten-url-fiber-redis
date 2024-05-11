package helpers

import (
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
)

func EnforceHTTP(url string) string {

	if url[:4] != "http" {
		return "http://" + url
	}
	return url

}

func RemoveDomainError(url string) bool {

	if url == os.Getenv("DOMAIN") {
		return false
	}

	// basically this functions removes all the commonly found
	// prefixes from URL such as http, https, www
	// then checks of the remaining string is the DOMAIN itself

	newURL := strings.Replace(url, "http://", "", 1)
	newURL = strings.Replace(newURL, "http://", "", 1)
	newURL = strings.Replace(newURL, "www.", "", 1)
	newURL = strings.Split(newURL, "/")[0]

	if newURL == os.Getenv("DOMAIN") {
		return false
	}

	return true
}

func GetClientIP(c *fiber.Ctx) string {
	clientIP := c.Get(fiber.HeaderXForwardedFor)
	if clientIP == "" {
		clientIP = c.IP()
	}
	return clientIP
}
