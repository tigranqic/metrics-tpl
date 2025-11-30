# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

# run test
 - make test ITERATION=1 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=2A SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=2B SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=3A SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=3B SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=4 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=5 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=6 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=7 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=8 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=9 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json
 - make test ITERATION=10A SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json DATABASE_DSN="postgres://postgres:postgres@localhost:15449/metrics-tpl?sslmode=disable"
 - make test ITERATION=10B SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json DATABASE_DSN="postgres://postgres:postgres@localhost:15448/metrics-tpl?sslmode=disable" 
 - make test ITERATION=11 SOURCE_PATH=. SERVER_PORT=8080 FILE_STORAGE_PATH=metrics.json DATABASE_DSN="postgres://postgres:postgres@localhost:15449/metrics-tpl?sslmode=disable"

# migration
make migrate-down DATABASE_DSN="postgres://postgres:postgres@localhost:15449/metrics-tpl?sslmode=disable"
