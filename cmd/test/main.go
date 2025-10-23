package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bojand/gotodo/components"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/redis/go-redis/v9"
)

// User represents a user in the system
type User struct {
	ID        string
	Name      string
	Passwd    string
	DeletedAt *time.Time // For soft delete
}

// Todo represents a single todo item
type Todo struct {
	ID        string
	Content   string
	Completed bool
}

// App state
type App struct {
	store *session.Store
	todos map[string][]Todo
	redis *redis.Client
	db    *DB // Database connection
}

func main() {
	// Initialize Redis client (still needed for user validation and todos)
	redisAddr := os.Getenv("REDIS_ADDRESS")
	redisPass := os.Getenv("REDIS_PASSWORD")
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPass,
	})

	// Initialize session storage with in-memory store
	store := session.New()

	// Initialize database (in-memory for demo)
	db := NewDB()

	// Initialize app
	app := &App{
		store: store,
		todos: make(map[string][]Todo),
		redis: rdb,
		db:    db,
	}

	// Initialize Fiber
	fiberApp := fiber.New(fiber.Config{
		Views: components.NewHTMLRenderer(),
	})

	// Serve static Tailwind CSS
	fiberApp.Static("/static", "./static")

	// Routes
	fiberApp.Get("/login", app.handleLoginPage)
	fiberApp.Post("/login", app.handleLogin)
	fiberApp.Get("/logout", app.handleLogout)

	// Protected routes with auth middleware
	protected := fiberApp.Group("/", app.authMiddleware)
	protected.Get("/", app.handleIndex)
	protected.Post("/todos", app.handleAddTodo)
	protected.Put("/todos/:id/toggle", app.handleToggleTodo)
	protected.Delete("/todos/:id", app.handleDeleteTodo)

	// Start server
	log.Fatal(fiberApp.Listen(":3000"))
}

// authMiddleware checks if the user is authenticated and valid
func (app *App) authMiddleware(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return c.Redirect("/login")
	}

	userID := sess.Get("userID")
	if userID == nil {
		return c.Redirect("/login")
	}

	// Check Redis for user validity
	ctx := context.Background()
	valid, err := app.redis.Get(ctx, "user:valid:"+userID.(string)).Result()
	if err == redis.Nil {
		// Not in Redis, check DB
		user, err := app.db.GetUser(userID.(string))
		if err != nil || user == nil || user.DeletedAt != nil {
			// User not found or soft-deleted, clear session
			sess.Destroy()
			return c.Redirect("/login")
		}
		// Cache in Redis (valid for 1 hour)
		if err := app.redis.Set(ctx, "user:valid:"+userID.(string), "1",
			time.Hour).Err(); err != nil {
			log.Printf("Error caching user in Redis: %v", err)
		}
	} else if err != nil {
		log.Printf("Redis error: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
	} else if valid != "1" {
		// Invalid user in Redis, clear session
		sess.Destroy()
		return c.Redirect("/login")
	}

	return c.Next()
}

func (app *App) handleLoginPage(c *fiber.Ctx) error {
	return c.Render("login", nil, "layouts/main")
}

func (app *App) handleLogin(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return err
	}

	name := c.FormValue("name")
	passwd := c.FormValue("passwd")
	user, err := app.db.GetUserByName(name)
	if err != nil || user == nil || user.Passwd != passwd || user.DeletedAt != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid credentials")
	}

	// Set user ID in session
	sess.Set("userID", user.ID)
	if err := sess.Save(); err != nil {
		return err
	}

	// Cache user validity in Redis
	ctx := context.Background()
	if err := app.redis.Set(ctx, "user:valid:"+user.ID, "1",
		time.Hour).Err(); err != nil {
		log.Printf("Error caching user in Redis: %v", err)
	}

	return c.Redirect("/")
}

func (app *App) handleLogout(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return err
	}

	userID := sess.Get("userID")
	if userID != nil {
		// Clear user validity from Redis
		ctx := context.Background()
		if err := app.redis.Del(ctx, "user:valid:"+userID.(string)).Err(); err != nil {
			log.Printf("Error deleting user from Redis: %v", err)
		}
	}

	sess.Destroy()
	return c.Redirect("/login")
}

func (app *App) handleIndex(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return err
	}

	userID := sess.Get("userID").(string)
	todos := app.todos[userID]

	// Save todos to Redis
	ctx := context.Background()
	if err := app.redis.Set(ctx, "todos:"+userID, todos,
		24*time.Hour).Err(); err != nil {
		log.Printf("Error saving to Redis: %v", err)
	}

	return c.Render("index", fiber.Map{
		"Todos": todos,
	}, "layouts/main")
}

func (app *App) handleAddTodo(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return err
	}

	userID := sess.Get("userID").(string)
	content := c.FormValue("content")
	if content == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Content is required")
	}

	todo := Todo{
		ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
		Content: content,
	}
	app.todos[userID] = append(app.todos[userID], todo)

	// Update Redis
	ctx := context.Background()
	if err := app.redis.Set(ctx, "todos:"+userID, app.todos[userID],
		24*time.Hour).Err(); err != nil {
		log.Printf("Error saving to Redis: %v", err)
	}

	// Return the new todo item HTML
	return components.TodoItem(todo).Render(c)
}

func (app *App) handleToggleTodo(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return err
	}

	userID := sess.Get("userID").(string)
	id := c.Params("id")

	for i, todo := range app.todos[userID] {
		if todo.ID == id {
			app.todos[userID][i].Completed = !todo.Completed
			// Update Redis
			ctx := context.Background()
			if err := app.redis.Set(ctx, "todos:"+userID, app.todos[userID],
				24*time.Hour).Err(); err != nil {
				log.Printf("Error saving to Redis: %v", err)
			}
			return components.TodoItem(app.todos[userID][i]).Render(c)
		}
	}

	return c.Status(fiber.StatusNotFound).SendString("Todo not found")
}

func (app *App) handleDeleteTodo(c *fiber.Ctx) error {
	sess, err := app.store.Get(c)
	if err != nil {
		return err
	}

	userID := sess.Get("userID").(string)
	id := c.Params("id")

	for i, todo := range app.todos[userID] {
		if todo.ID == id {
			app.todos[userID] = append(app.todos[userID][:i], app.todos[userID][i+1:]...)
			// Update Redis
			ctx := context.Background()
			if err := app.redis.Set(ctx, "todos:"+userID, app.todos[userID],
				24*time.Hour).Err(); err != nil {
				log.Printf("Error saving to Redis: %v", err)
			}
			return c.SendString("")
		}
	}

	return c.Status(fiber.StatusNotFound).SendString("Todo not found")
}
