package handlers

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"todo/database"
	"todo/models"

	"github.com/gofiber/fiber/v2"
)

// Функция извлечения контекста из запроса
// C помощью нее получаем подключение к БД
func GetLocal[T any](c *fiber.Ctx, key string) T {
	return c.Locals(key).(T)
}

// GET /tasks handler
func GetTasks(c *fiber.Ctx) error {
	db := GetLocal[*database.DB](c, "db")

	conn, err := db.Pool.Acquire(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": "Unable to acquire a database connection: " + err.Error(),
		})
	}
	defer conn.Release()

	rows, err := conn.Query(context.Background(), "SELECT * FROM tasks")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": "Unable to SELECT records: " + err.Error(),
		})
	}
	defer rows.Close()

	// В данный слайс сохраним полученные из БД записи
	tasks := []models.Task{}
	for rows.Next() {
		var task models.Task
		err := rows.Scan(
			&task.ID, &task.Title, &task.Description, &task.Status, &task.Created_at, &task.Updated_at,
		)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
				"Error": err.Error(),
			})
		}
		tasks = append(tasks, task)
	}

	return c.Status(fiber.StatusOK).JSON(tasks)
}

// POST /tasks handler
func CreateTask(c *fiber.Ctx) error {
	db := GetLocal[*database.DB](c, "db")

	// Анмаршалим JSON из Body в экземпляр структуры
	task := new(models.Task)
	if err := c.BodyParser(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"Error": "Bad request: " + err.Error(),
		})
	}

	// Валидируем полученные из запроса данные:
	// в случае ошибки, возвращаем 400-й ответ пользователю
	if err := models.ModelValidator(task); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"Error": "Bad request " + err.Error(),
		})
	}

	// В случае, если валидация оказалась успешной, добавляем запись в БД
	conn, err := db.Pool.Acquire(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": "Unable to acquire a database connection: " + err.Error(),
		})
	}
	defer conn.Release()

	ct, err := conn.Exec(context.Background(),
		"INSERT INTO tasks (title, description, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)",
		task.Title, task.Description, task.Status, task.Created_at, task.Updated_at,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": "Unable to INSERT in database: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(struct{ Inserted int }{Inserted: int(ct.RowsAffected())})
}

// Вспомогательная функция для обработчика PUT /tasks/:id
// Используя рефлексию, динамически работаем с данными, пришедшими из запроса,
// Присваиваем их в экземпляр структуры task, валидируем структуру валидатором
// В результате возвращаем слайс имен полей в БД, которые требуется обновить, а также
// слайс значений для обновления. Если валидация пройдена с ошибкой, либо же пользователь
// передал некорректные данные, не доступные для обновления (например, хочет обновить PRIMARY KEY),
// функция вернет ошибку
func updater(task any, rawData map[string]any) ([]string, []any, error) {
	val := reflect.ValueOf(task).Elem()
	keys := []string{}
	values := []any{}
	var err error

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)
		key := strings.ToLower(typeField.Name)
		if v, exists := rawData[key]; exists {
			// первичный ключ не обновляется, вернем ошибку
			if key == "id" {
				return nil, nil, fmt.Errorf("primary key cannot be updated for an existing record")
			}
			switch typeField.Type.Kind() {
			case reflect.Int:
				if i, ok := v.(int); ok {
					valueField.Set(reflect.ValueOf(int(i)))
					keys = append(keys, key)
					values = append(values, i)
				} else {
					return nil, nil, fmt.Errorf("invalid %s field type", key)
				}
			case reflect.String:
				if s, ok := v.(string); ok {
					valueField.SetString(string(s))
					keys = append(keys, key)
					values = append(values, s)
				} else {
					return nil, nil, fmt.Errorf("invalid %s field type", key)
				}
			case reflect.Pointer:
				if sPtr, ok := v.(string); ok {
					valueField.Set(reflect.ValueOf(&sPtr))
					keys = append(keys, key)
					values = append(values, sPtr)
				} else {
					return nil, nil, fmt.Errorf("invalid %s field type", key)
				}
			}
			// Если в запросе не было поля title, добавим в экземпляр структуры заглушку
			// для этого поля, чтобы пройти валидацию
		} else if typeField.Name == "Title" {
			valueField.SetString(string("Dummy"))
		}
	}

	if task, ok := task.(*models.Task); ok {
		err = models.ModelValidator(task)
	}
	return keys, values, err
}

// PUT /tasks/:id handler
func UpdateTask(c *fiber.Ctx) error {
	db := GetLocal[*database.DB](c, "db")

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"Error": "Bad request: " + err.Error(),
		})
	}

	rawData := make(map[string]any)
	if err := c.BodyParser(&rawData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"Error": "Bad request: " + err.Error(),
		})
	}

	// тут проверяем, имеет ли смысл идти дальше и обновлять запись в БД
	// или вернем ошибку пользователю
	keys, vals, err := updater(new(models.Task), rawData)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"Error": "Bad request: " + err.Error(),
		})
	}

	conn, err := db.Pool.Acquire(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": "Unable to acquire a database connection: " + err.Error(),
		})
	}
	defer conn.Release()

	// С помощью этого цикла конструируем запрос в БД из ключей и значений, полученных
	// из вспомогательной функции. Используется байтовый буффер вместо строк для более эффективной
	// работы
	query := bytes.NewBuffer([]byte("UPDATE tasks SET "))
	var i int
	var key string
	for i, key = range keys {
		query.Write([]byte(fmt.Sprintf("%s = $%d", key, i+1)))
		if i != len(keys)-1 {
			query.Write([]byte(", "))
		}
	}
	query.Write([]byte(fmt.Sprintf(" WHERE id = %d", id)))

	ct, err := conn.Exec(context.Background(), query.String(), vals...)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": err.Error(),
		})
	}
	if ct.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(map[string]any{
			"Error": "Record not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(struct{ Updated int64 }{Updated: ct.RowsAffected()})
}

// DELETE /tasks/:id handler
func DeleteTask(c *fiber.Ctx) error {
	db := GetLocal[*database.DB](c, "db")

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(map[string]any{
			"Error": "Bad request: " + err.Error(),
		})
	}

	conn, err := db.Pool.Acquire(context.Background())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": "Unable to acquire a database connection: " + err.Error(),
		})
	}
	defer conn.Release()

	ct, err := conn.Exec(context.Background(),
		"DELETE FROM tasks WHERE id = $1", id,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(map[string]any{
			"Error": err.Error(),
		})
	}
	if ct.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(map[string]any{
			"Error": "Record not found",
		})
	}

	return c.Status(fiber.StatusOK).JSON(struct{ Affected int64 }{Affected: ct.RowsAffected()})
}
