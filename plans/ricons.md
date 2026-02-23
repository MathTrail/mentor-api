Вот финальный, максимально полный и профессионально оформленный список бейджей для твоего README.md.

Я разбил их на логические строки, чтобы они не превращались в кашу и хорошо смотрелись как на десктопе, так и на мобильных устройствах.

Скопируй этот блок в начало README:
Markdown
# mentor-api

[![CI](https://github.com/MathTrail/mentor-api/actions/workflows/ci.yml/badge.svg)](https://github.com/MathTrail/mentor-api/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/MathTrail/mentor-api)](https://goreportcard.com/report/github.com/MathTrail/mentor-api)
[![codecov](https://codecov.io/gh/MathTrail/mentor-api/branch/main/graph/badge.svg)](https://codecov.io/gh/MathTrail/mentor-api)
[![Go Version](https://img.shields.io/github/go-mod/go-version/MathTrail/mentor-api)](https://github.com/MathTrail/mentor-api/blob/main/go.mod)
[![Go Reference](https://pkg.go.dev/badge/github.com/MathTrail/mentor-api.svg)](https://pkg.go.dev/github.com/MathTrail/mentor-api)

[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-326CE5?style=for-the-badge&logo=kubernetes)](./deploy/charts)
[![Dapr](https://img.shields.io/badge/Dapr-Enabled-007ACC?style=for-the-badge&logo=dapr)](https://dapr.io/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791?style=for-the-badge&logo=postgresql)](https://www.postgresql.org/)

[![API Docs](https://img.shields.io/badge/API_Docs-Swagger-85EA2D?style=for-the-badge&logo=swagger)](https://MathTrail.github.io/mentor-api/)
[![Tracing](https://img.shields.io/badge/Tracing-OTel-000000?style=for-the-badge&logo=opentelemetry)](https://opentelemetry.io/)
[![Profiling](https://img.shields.io/badge/Profiling-Pyroscope-FF7800?style=for-the-badge&logo=pyroscope)](https://pyroscope.io/)

---
Что в этом списке (краткий справочник):
Верхняя строка (Лаконичные бейджи):

CI: Твой "зеленый свет" от GitHub Actions.

Go Report Card: Твоя оценка за чистоту кода (A/A+).

Codecov: Процент покрытия тестами.

Go Version: Автоматически подтягивается из go.mod.

Go Reference: Прямая ссылка на документацию функций.

Средняя строка (Инфраструктурные плашки):

Kubernetes: Ссылка на твои Helm-чарты (замени ./deploy/charts на реальный путь).

Dapr: Гордо заявляем об использовании современной Runtime-платформы.

PostgreSQL: Указываем конкретную версию БД.

Нижняя строка (Инструменты):

Swagger: Самый заметный бейдж для тех, кто хочет сразу пощупать API.

OTel & Pyroscope: Показывают, что проект полностью готов к мониторингу в продакшене.

Твой следующий шаг
Чтобы большинство этих иконок "ожили", тебе нужен файл конфигурации для GitHub Actions. Без него бейдж CI будет показывать no status, а Codecov будет пустым.

Хочешь, я сгенерирую файл .github/workflows/ci.yml, который сразу настроит тесты, линтер и отправку покрытия?