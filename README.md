# 🌱 Огородник
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Svelte](https://img.shields.io/badge/Frontend-Svelte%2BVite-FF3E00?style=flat&logo=svelte)](https://svelte.dev/)
[![SQLite](https://img.shields.io/badge/Database-SQLite-003B57?style=flat&logo=sqlite)](https://www.sqlite.org/)
[![CI/CD](https://github.com/unlaudd/gardener/actions/workflows/ci-cd.yml/badge.svg)](https://github.com/unlaudd/gardener/actions)
[![Tests](https://img.shields.io/badge/Tests-23%20passed-brightgreen)](./gardener/internal/)
[![Lint](https://img.shields.io/badge/Lint-golangci%2Beslint-blue)](https://golangci-lint.run/)
[![License](https://img.shields.io/badge/License-MIT-blue)](LICENSE)

Локальное веб-приложение для учёта семян, агротехники и фотофиксации растений. Работает полностью офлайн, хранит данные в SQLite, поставляется в виде одного самодостаточного бинарного файла.

## Возможности
- Полноценный CRUD учётных записей растений
- Галерея фото с поддержкой нескольких изображений, выбором обложки и удалением
- Поиск по виду, тегам и серийному номеру с подсветкой и автопрокруткой
- Экспорт данных в CSV или TXT с настраиваемым набором полей
- Адаптивный интерфейс (таблица / галерея карточек)
- Офлайн-работа без внешних зависимостей и облачных сервисов

## Технологический стек
| Компонент | Технология                               |
|-----------|------------------------------------------|
| Backend   | Go 1.22+, `net/http`, `database/sql`     |
| Database  | SQLite (`modernc.org/sqlite`, чистый Go) |
| Frontend  | Svelte 5, Vite, vanilla CSS              |
| Сборка    | `//go:embed`, GoReleaser, Makefile       |
| CI/CD     | GitHub Actions, golangci-lint, ESLint    |

## Быстрый старт
```bash
# 1. Установка зависимостей
cd gardener && make install-deps

# 2. Сборка приложения
make build

# 3. Запуск
./gardener
# Приложение автоматически откроет браузер по адресу http://127.0.0.1:8080
```

## Структура проекта

```
.
├── .github/workflows/ci-cd.yml   # Конфигурация CI/CD пайплайна
├── .golangci.yml                 # Настройки Go-линтера
├── .gitignore                    # Правила игнорирования файлов
├── .goreleaser.yaml              # Конфигурация кросс-компиляции и релизов
├── README.md                     # Текущий файл
└── gardener/                     # Исходный код приложения
    ├── main.go                   # Точка входа, инициализация сервера и роутинг
    ├── Makefile                  # Цели сборки, тестов, линтинга и запуска
    ├── frontend/                 # Svelte/Vite фронтенд
    │   ├── src/                  # Исходные файлы компонентов
    │   ├── eslint.config.js      # Конфигурация ESLint
    │   └── package.json          # Зависимости и скрипты
    └── internal/
        ├── api/                  # HTTP-обработчики и тесты API
        └── db/                   # Работа с SQLite, хранение фото и тесты БД
```

## CI/CD и релизы
Пайплайн автоматизирован через GitHub Actions:

1. Continuous Integration: Запускается при каждом push и pull_request. Выполняет линтинг (make lint) и тесты (make test).
2. Continuous Delivery: Запускается при создании тега формата v* (например, v1.0.0). Использует GoReleaser для:
   * Кросс-компиляции под Linux, Windows, macOS (amd64/arm64)
   * Автоматической сборки фронтенда через pre-хук
   * Генерации CHANGELOG.md на основе истории коммитов
   * Публикации архивов в GitHub Releases

```bash
# Создание и публикация релиза
git tag v1.0.0
git push origin main --tags
```

## Разработка и утилиты
Все операции унифицированы через Makefile:

| Команда           | Описание                                    |
|-------------------|---------------------------------------------|
| make install-deps | Установка Go и Node.js зависимостей         |
| make build        | Полная сборка бинарного файла               |
| make run          | Сборка и локальный запуск                   |
| make test         | Запуск тестов с проверкой гонок и покрытием |
| make lint         | Статический анализ Go и JS/Svelte кода      |
| make fmt          | Автоформатирование кода (gofmt + prettier)  |
| make clean        | Очистка артефактов сборки и кэшей           |

## Документация кода
Все пакеты, экспортированные функции, структуры и тесты покрыты комментариями в формате godoc. 
Документация следует стандартам Go: описывает назначение, параметры, возвращаемые значения и побочные эффекты.
Локальный просмотр доступен через: ```godoc -http=:6060```

## Лицензия
MIT. Проект распространяется «как есть» без гарантий. См. файл LICENSE для деталей.