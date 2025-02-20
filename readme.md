# TODO list Go + Fiber + PostgreSQL (через pgx)

Запуск: 
```
docker compose up -d
```
Приложение запускается на 8080 порту.
Обязательным заголовком в запросах является "Content-Type":"application/json; charset=utf-8"
Body запроса - в JSON.

* POST      /tasks – создание задачи.
* GET       /tasks – получение списка всех задач.
* PUT       /tasks/id – обновление задачи.
* DELETE    /tasks/id – удаление задачи.
