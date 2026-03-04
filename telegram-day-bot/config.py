# -*- coding: utf-8 -*-
"""
Настройки из .env. Один источник — смена провайдера в одну переменную.
"""
import os
from pathlib import Path

def _env(key: str, default: str = "") -> str:
    return os.environ.get(key, default).strip()


# Telegram
BOT_TOKEN = _env("SHOZA_BOT_TOKEN")
CHANNEL_ID = _env("SHOZA_CHANNEL_ID")  # @shoza_den или -1001234567890
ADMIN_CHAT_ID = _env("SHOZA_ADMIN_CHAT_ID")  # твой chat_id для выбора вариантов (этап 2)

# Режим публикации (этап 1: simple_auto; этап 2: approval / approval_with_timeout)
PUBLISH_MODE = _env("SHOZA_PUBLISH_MODE", "simple_auto")  # simple_auto | approval | approval_with_timeout
APPROVAL_TIMEOUT_MINUTES = int(_env("SHOZA_APPROVAL_TIMEOUT_MINUTES", "120"))

# AI: какой провайдер использовать (simple | yandex | deepseek | local)
AI_PROVIDER = _env("SHOZA_AI_PROVIDER", "simple")

# Провайдеры — свои ключи (для этапа 2)
YANDEX_API_KEY = _env("SHOZA_YANDEX_API_KEY")
YANDEX_FOLDER_ID = _env("SHOZA_YANDEX_FOLDER_ID")
DEEPSEEK_API_KEY = _env("SHOZA_DEEPSEEK_API_KEY")
LOCAL_AI_BASE_URL = _env("SHOZA_LOCAL_AI_BASE_URL", "http://localhost:11434")
LOCAL_AI_MODEL = _env("SHOZA_LOCAL_AI_MODEL", "")

# БД (SQLite по умолчанию)
DATA_DIR = Path(_env("SHOZA_DATA_DIR", str(Path(__file__).parent / "data")))
DB_PATH = str(DATA_DIR / "posts.db")

# Расписание: время утреннего поста (UTC или локальное — уточни при деплое)
POST_HOUR = int(_env("SHOZA_POST_HOUR", "7"))
POST_MINUTE = int(_env("SHOZA_POST_MINUTE", "0"))
