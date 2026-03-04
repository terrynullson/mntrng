# -*- coding: utf-8 -*-
"""
Точка входа. Этап 1: по расписанию генерируем один текст (из заготовок),
отправляем в канал, пишем в БД. Этап 2: здесь же будет запуск логики
с 5 вариантами и тайм-аутом.
"""
import asyncio
import logging
import sys
from datetime import date, datetime, timedelta, time

from dotenv import load_dotenv
from telegram import Bot
from telegram.ext import Application, CommandHandler, ContextTypes

load_dotenv()

import config
from ai import get_generator
from db import repository

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
log = logging.getLogger("shoza")


# Дни недели по-русски
WEEKDAYS = [
    "понедельник", "вторник", "среда", "четверг", "пятница", "суббота", "воскресенье"
]


def today_weekday_ru() -> str:
    return WEEKDAYS[date.today().weekday()]


async def cmd_status(update, context: ContextTypes.DEFAULT_TYPE) -> None:
    """Ответ только тому, чей chat_id совпадает с SHOZA_ADMIN_CHAT_ID."""
    if not update.effective_chat or not update.message:
        return
    if str(update.effective_chat.id) != config.ADMIN_CHAT_ID:
        return
    await update.message.reply_text(f"Я в порядке, сегодня {today_weekday_ru()}.")


async def run_today_post() -> None:
    """Один пост на сегодня: сгенерировать текст, отправить в канал, сохранить в БД."""
    today = date.today()
    day_name = today_weekday_ru()

    # Уже постили на сегодня?
    existing = await repository.get_post_by_date(config.DB_PATH, today)
    if existing:
        log.info("Пост на сегодня уже есть, пропуск. post_date=%s", today)
        return

    generator = get_generator(config.AI_PROVIDER, config=config)
    variants = await generator.generate_variants(day_name, count=1)
    text = (variants[0] if variants else f"сегодня {day_name} если что").strip()

    bot = Bot(token=config.BOT_TOKEN)
    await bot.send_message(chat_id=config.CHANNEL_ID, text=text)
    sent_at = datetime.utcnow().isoformat() + "Z"

    await repository.insert_post(
        config.DB_PATH,
        post_date=today,
        day_of_week=day_name,
        message_text=text,
        approval_status=repository.APPROVAL_MANUAL_FULL,
        sent_at=sent_at,
    )
    log.info("Пост отправлен: %s", text[:50])


def next_run_at() -> datetime:
    """Следующий запуск — сегодня или завтра в POST_HOUR:POST_MINUTE (локальное время)."""
    now = datetime.now()
    target = now.replace(hour=config.POST_HOUR, minute=config.POST_MINUTE, second=0, microsecond=0)
    if now >= target:
        target += timedelta(days=1)
    return target


async def scheduler_loop() -> None:
    """Бесконечный цикл: ждём следующего времени поста и выполняем run_today_post."""
    await repository.ensure_db(config.DB_PATH)
    while True:
        at = next_run_at()
        log.info("Следующий пост запланирован на %s", at)
        delay = (at - datetime.now()).total_seconds()
        if delay > 0:
            await asyncio.sleep(delay)
        try:
            await run_today_post()
        except Exception as e:
            log.exception("Ошибка при отправке поста: %s", e)


async def daily_job(context) -> None:
    """Ежедневный пост по расписанию (для режима с Application)."""
    try:
        await run_today_post()
    except Exception as e:
        log.exception("Ошибка при отправке поста: %s", e)


def main() -> None:
    if not config.BOT_TOKEN or not config.CHANNEL_ID:
        log.error("Задайте SHOZA_BOT_TOKEN и SHOZA_CHANNEL_ID в .env")
        sys.exit(1)
    config.DATA_DIR.mkdir(parents=True, exist_ok=True)
    # Запуск один раз (для теста): python run.py once
    if len(sys.argv) > 1 and sys.argv[1].strip().lower() == "once":
        asyncio.run(_run_once())
        return
    # С админом: бот слушает команды и по расписанию постит (одна точка входа)
    if config.ADMIN_CHAT_ID:
        _run_with_bot()
    else:
        asyncio.run(scheduler_loop())


def _run_with_bot() -> None:
    """Polling + команда /status для админа + ежедневный пост по расписанию."""
    app = (
        Application.builder()
        .token(config.BOT_TOKEN)
        .post_init(_post_init)
        .build()
    )
    app.add_handler(CommandHandler("status", cmd_status))
    app.job_queue.run_daily(daily_job, time=time(config.POST_HOUR, config.POST_MINUTE))
    log.info("Бот запущен. Команда /status в личку — проверка (только для SHOZA_ADMIN_CHAT_ID).")
    app.run_polling(drop_pending_updates=True)


async def _post_init(app: Application) -> None:
    await repository.ensure_db(config.DB_PATH)


async def _run_once() -> None:
    """Отправить один пост прямо сейчас (для проверки механики)."""
    await repository.ensure_db(config.DB_PATH)
    await run_today_post()


if __name__ == "__main__":
    main()
