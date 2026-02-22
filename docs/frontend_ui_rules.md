# Frontend UI Rules (Contract)

FrontendAgent обязан:
- Следовать PROMPTS/ui_style_guide.md без отклонений.
- Использовать TailwindCSS.
- Использовать shadcn/ui как базовые компоненты (Button, Table, Badge, Dropdown, Dialog, Tabs).
- Использовать Framer Motion для микро-анимаций и переходов.
- Не добавлять дизайнерские элементы, не описанные в ui_style_guide, без ADR.

Запрещено:
- Менять API контракты без согласования с MasterAgent.
- Имплементировать бизнес-логику мониторинга на фронте.
- Использовать “самописные” компоненты вместо shadcn, если есть аналог.

Definition of Done (для любой UI-задачи):
- Реализованы states: loading (skeleton), empty, error
- Стили соответствуют гайду (flat, минимализм)
- Таблицы: сортировка/фильтры/поиск (минимум search + status)
- Переиспользование компонентов (no copy-paste стилей)
- 1 задача = 1 коммит
