-- История постов канала «День, если что»
-- Статусы одобрения: различаем ручное одобрение, тайм-аут и полностью ручной пост

CREATE TABLE IF NOT EXISTS posts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_date DATE NOT NULL,
    day_of_week TEXT NOT NULL,
    message_text TEXT NOT NULL,
    approval_status TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    sent_at TEXT,
    UNIQUE(post_date)
);

-- approval_status:
--   approved_manual  — я выбрал один из вариантов (или отредактировал)
--   timeout          — не успел ответить, ушла заглушка
--   manual_full      — пост написан/выбран полностью вручную (в т.ч. этап 1 без выбора)
--   rejected         — отклонил все варианты (пост не отправлялся или отправили заглушку — можно уточнить логику)

CREATE INDEX IF NOT EXISTS idx_posts_post_date ON posts(post_date);
CREATE INDEX IF NOT EXISTS idx_posts_approval_status ON posts(approval_status);
