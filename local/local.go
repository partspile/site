package local

import "github.com/gofiber/fiber/v2"

func GetUserID(c *fiber.Ctx) int {
	userID, _ := c.Locals("userID").(int)
	return userID
}

func SetUserID(c *fiber.Ctx, userID int) {
	c.Locals("userID", userID)
}

func SetUserName(c *fiber.Ctx, userName string) {
	c.Locals("userName", userName)
}

func GetUserName(c *fiber.Ctx) string {
	userName, _ := c.Locals("userName").(string)
	return userName
}
