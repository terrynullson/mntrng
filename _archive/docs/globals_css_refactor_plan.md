# Анализ и план рефакторинга globals.css

## Текущее состояние

- **Файл:** `web/app/globals.css` (~2550 строк)
- **Подключение:** один раз в `web/app/layout.tsx` через `import "./globals.css"`.
- **Структура:** один монолитный файл без разбиения; порядок правил определяет каскад.

## Логические блоки (по порядку)

| Блок | Примерные строки | Содержимое |
|------|------------------|------------|
| **tokens** | 1–82 | `:root`, `[data-theme="dark"]` переменные + несколько тёмных переопределений (nav, button, status, skeleton, form disabled) |
| **base** | 84–112 | Reset: `*`, `html`/`body`, `a`, `:focus-visible` |
| **public** | 113–132 | `.public-root`, `.public-theme-toggle` |
| **theme-toggle** | 133–178 | Переключатель «лампочка»: `.theme-toggle`, `.theme-toggle__lightbulb*`, reduced-motion |
| **theme-switcher-grid (мёртвый)** | 179–346 | Старый переключатель dottereldesign/CodePen. В коде не используется: в `theme-toggle-button.tsx` соответствующая кнопка закомментирована. **Можно удалить.** |
| **auth** | 348–668 | `.auth-page`, `.auth-card`, blobs, orbs, keyframes, noise, `.pending-card` |
| **secure-shell** | 670–814 | `.secure-shell`, sidebar, nav, topbar, user-menu, `.protected-loading` |
| **layout** | 816–902 | `.panel`, `.page-header`, `.overview-grid`, `.landing-primary-*`, `.filters-grid` |
| **forms** | 904–982 | `.form-field`, input/select/textarea |
| **tables** | 983–… | `.table-wrap`, `table`, `th`/`td`, `.stream-link` |
| **buttons** | разбросано | `.button-primary`, `.button-secondary`, `.button-danger`, `.button-ghost` |
| **components** | разбросано | Status badges, skeleton, settings-card, analytics, users, companies и т.д. |
| **hub** | ~1850–2550 | `.hub-page`, `.hub-content`, `.hub-grid`, `.module-card`, `.hub-bg-blobs`, media queries, `.user-pill`, `.section-root` |

## Рекомендации

### 1. Быстрый выигрыш: удалить мёртвый код

- Удалить блок **.theme-switcher-grid** (строки 179–346, ~168 строк).
- В `theme-toggle-button.tsx` уже используется только кнопка с классом `.theme-toggle` (лампочка); вторая кнопка с `theme-switcher-grid` закомментирована.
- **Эффект:** уменьшение размера файла без изменения поведения.

### 2. Разбиение на части через @import (один бандл)

- Создать каталог `web/app/styles/` (или `web/styles/`).
- Вынести логические блоки в отдельные файлы, например:
  - `tokens.css` — переменные и тёмная тема
  - `base.css` — reset, body, focus, ссылки
  - `theme-toggle.css` — только лампочка
  - `auth.css` — auth-page, auth-card, blobs, orbs, keyframes
  - `secure-shell.css` — secure-*, protected-loading
  - `layout.css` — panel, page-header, overview, landing, filters-grid
  - `forms.css` — form-field, inputs
  - `tables.css` — table-wrap, table, stream-link
  - `buttons.css` — все .button-*
  - `components.css` — status, skeleton, settings, analytics, users, companies и прочие общие компоненты
  - `hub.css` — все .hub-*, .module-card, hub media queries, user-pill, section-root
- В `globals.css` оставить только последовательность `@import "styles/...";` в нужном порядке (tokens → base → … → hub).
- **Эффект:** тот же объём бандла, но удобная навигация и правки по зонам ответственности.

### 3. Опционально: загрузка hub-стилей по маршруту

- Подключать `hub.css` не из `globals.css`, а из `web/app/hub/layout.tsx` (например, `import "@/app/styles/hub.css"`).
- Тогда стили Hub попадут в отдельный chunk и подгрузятся только при заходе на `/hub`.
- **Эффект:** чуть меньший начальный CSS для пользователей, которые не открывают Hub.

### 4. Чего не делать без отдельного решения

- **CSS Modules** — многие классы завязаны на глобальные имена (BEM-подобные), массовый перенос в модули потребует правок разметки и риска поломок.
- **Tailwind** — смена подхода к стилям; выходит за рамки «привести в порядок globals.css».

## Итог

- **Минимум:** удалить блок `.theme-switcher-grid` (~168 строк).
- **Рекомендуемый шаг:** разбить `globals.css` на части в `web/app/styles/` с @import, сохранив один точку входа в `layout.tsx`.
- **По желанию:** вынести `hub.css` в layout маршрута `/hub` для отложенной загрузки.
