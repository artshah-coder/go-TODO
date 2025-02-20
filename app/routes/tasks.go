package routes

import (
	"todo/handlers"

	"github.com/gofiber/fiber/v2"
)

func RegisterTaskRoutes(app *fiber.App) {
	app.Get("/tasks", handlers.GetTasks)
	app.Post("/tasks", handlers.CreateTask)
	app.Put("/tasks/:id", handlers.UpdateTask)
	app.Delete("/tasks/:id", handlers.DeleteTask)
}
