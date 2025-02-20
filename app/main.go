package main

import (
	"fmt"
	"log"
	"strings"
	"todo/database"
	"todo/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

// Функция для установки контекста.
// В контексте будем передавать подключение к БД
func SetLocal[T any](c *fiber.Ctx, key string, value T) {
	c.Locals(key, value)
}

func main() {
	// Подключаемся к БД
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Error connect to database: %v", err)
	}
	defer db.Pool.Close()
	fmt.Println("Hey")

	app := fiber.New(fiber.Config{
		Prefork:       false,
		CaseSensitive: true,
		StrictRouting: true,
	})

	// Инициализирую middleware:
	// логирование, воссатновление после паники
	// а также middleware, устанавливающее контекст
	// и проверяющее заголовок Content-Type
	// приложение работает с JSON форматом
	app.Use(logger.New())
	app.Use(recover.New())
	app.Use(func(c *fiber.Ctx) error {
		cType := c.Get("Content-Type")
		if !strings.Contains(cType, "application/json") ||
			!strings.Contains(cType, "utf-8") {
			return c.Status(fiber.StatusUnsupportedMediaType).JSON(map[string]any{
				"Error": "Unsupported Media Type",
			})
		}
		SetLocal[*database.DB](c, "db", db)
		return c.Next()
	})

	// Регистрируем мультиплексор запросов
	routes.RegisterTaskRoutes(app)

	log.Fatal(app.Listen(":8080"))
}
