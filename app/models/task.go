package models

import (
	"time"

	"github.com/asaskevich/govalidator"
)

// Определяем модель данных
type Task struct {
	ID          int       `json:"id" valid:"range(0|2147483647)"`
	Title       string    `json:"title" valid:"required"`
	Description *string   `json:"description" valid:"optional"`
	Status      string    `json:"status" valid:"in(new|in_progress|done),optional"`
	Created_at  time.Time `json:"created_at"`
	Updated_at  time.Time `json:"updated_at"`
}

// Функция-валидатор полей структуры. Используется внешний пакет govalidator
// В случае типа time.Time валидируем вручную: если время не было передано или
// в полях структуры окажутся значения по умолчанию, в этом случае задаем значение
// time.Now() в соответсвии с ограничениями БД
func ModelValidator(task *Task) error {
	_, err := govalidator.ValidateStruct(task)
	if err != nil {
		return err
	}
	if task.Status == "" {
		task.Status = "new"
	}
	if task.Created_at.String() == "0001-01-01 00:00:00 +0000 UTC" {
		task.Created_at = time.Now()
	}
	if task.Updated_at.String() == "0001-01-01 00:00:00 +0000 UTC" {
		task.Updated_at = time.Now()
	}
	return nil
}
