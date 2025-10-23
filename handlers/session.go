package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/user"
)

var store = session.New()

func sessionGetUser(c *fiber.Ctx) *user.User {
	sess, err := store.Get(c)
	if err != nil {
		return nil
	}

	if userValue := sess.Get("user"); userValue != nil {
		return userValue.(*user.User)
	}

	return nil
}

func sessionSetUser(c *fiber.Ctx, user *user.User) error {
	sess, err := store.Get(c)
	if err != nil {
		return err
	}

	sess.Set("user", user)
	return sess.Save()
}

func sessionDestroy(c *fiber.Ctx) {
	sess, err := store.Get(c)
	if err == nil {
		sess.Destroy()
	}
}

func dbGetUser(userID int) *user.User {
	u, err := user.GetUser(userID)
	if err != nil || u.IsArchived() {
		return nil
	}
	return &u
}

func getUser(c *fiber.Ctx) *user.User {
	u, _ := c.Locals("user").(*user.User)
	return u
}

func setUser(c *fiber.Ctx, u *user.User) {
	c.Locals("user", u)
}

func SessionMiddleware(c *fiber.Ctx) error {
	// login stashes user in session store
	u := sessionGetUser(c)
	if u != nil {
		// found user, check redis for validity
		if redisUserInvalid(u.ID) {
			// not in redis, check db
			u = dbGetUser(u.ID)
			if u == nil {
				// not in db, destroy session
				sessionDestroy(c)
				return redirectToLogin(c)
			}
			// refresh user
			sessionSetUser(c, u)
			redisSetUserValid(u.ID)
		}
	}
	// stash user in context for use in next handlers
	setUser(c, u)
	return c.Next()
}
