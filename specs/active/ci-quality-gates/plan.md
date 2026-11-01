# Implementation Plan - ci-quality-gates

## Summary
Выполнить три пункта, одобренные пользователем.
## Approach
Добавить frontend и security jobs в существующий workflow, проверить их команды, затем настроить обязательные статусы dev через GitHub API.
## Execution Constraints
Не менять бизнес-логику, не пушить без отдельного указания, не добавлять в git посторонние локальные файлы. Не обходить найденные уязвимости continue-on-error.
## Tasks
- [x] Добавить тесты фронтенда и проверку зависимостей в workflow, настроить Chromium.
- [x] Выполнить тесты, сборку, аудит и валидацию workflow; устранить обнаруженные проблемы в согласованном объёме.
- [x] Настроить защиту dev и проверить результат чтением API.
- [x] Обновить README и записать результаты проверок.
## Validation Strategy
npm test, npm run build, полный Playwright, npm audit --audit-level=high, govulncheck ./..., actionlint, git diff --check, чтение защиты ветки GitHub API. Удалённый запуск нового workflow требует публикации изменений, которая отдельно не запрошена.
## Risks / Follow-ups
Результаты 01.11.2026: npm test — 6/6; npm run build — успешно; полный Playwright — 50/50; actionlint и git diff --check — без ошибок; govulncheck v1.9.0 — No vulnerabilities found; npm audit --audit-level=high — exit 0, два moderate-предупреждения React Router, high/critical нет. Старые E2E-ожидания обновлены под раздельную историю карты/копилки и текст простого режима. Изменений приложения и зависимостей нет. Защита dev включена и независимо прочитана API: strict=true, четыре обязательные проверки, admins=true, PR обязателен, force push/deletion запрещены.

Сканирование может обнаружить реальные уязвимости, требующие обновления зависимостей. Новые обязательные статусы появятся в GitHub только после публикации workflow. До публикации PR не сможет получить их; правила не ослаблять молча.
