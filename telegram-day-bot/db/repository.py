# -*- coding: utf-8 -*-
"""
Доступ к истории постов. Один слой — всё в одном файле для простоты.
"""
import aiosqlite
from datetime import date
from pathlib import Path
from typing import List, Optional

from .schema import init_db

APPROVAL_APPROVED = "approved_manual"
APPROVAL_TIMEOUT = "timeout"
APPROVAL_MANUAL_FULL = "manual_full"
APPROVAL_REJECTED = "rejected"


async def get_connection(db_path: str):
    return await aiosqlite.connect(db_path)


async def ensure_db(db_path: str) -> None:
    schema_path = Path(__file__).parent / "schema.sql"
    schema_path = schema_path.resolve()
    init_db(db_path, schema_path)


async def insert_post(
    db_path: str,
    post_date: date,
    day_of_week: str,
    message_text: str,
    approval_status: str,
    sent_at: Optional[str] = None,
) -> int:
    async with await get_connection(db_path) as conn:
        await conn.execute(
            """
            INSERT INTO posts (post_date, day_of_week, message_text, approval_status, sent_at)
            VALUES (?, ?, ?, ?, ?)
            """,
            (post_date.isoformat(), day_of_week, message_text, approval_status, sent_at),
        )
        await conn.commit()
        cur = await conn.execute("SELECT last_insert_rowid()")
        row = await cur.fetchone()
        return row[0] if row else 0


async def get_post_by_date(db_path: str, post_date: date) -> Optional[dict]:
    async with await get_connection(db_path) as conn:
        conn.row_factory = aiosqlite.Row
        cur = await conn.execute(
            "SELECT id, post_date, day_of_week, message_text, approval_status, created_at, sent_at FROM posts WHERE post_date = ?",
            (post_date.isoformat(),),
        )
        row = await cur.fetchone()
        return dict(row) if row else None


async def list_posts(
    db_path: str,
    limit: int = 100,
    approval_status: Optional[str] = None,
) -> List[dict]:
    async with await get_connection(db_path) as conn:
        conn.row_factory = aiosqlite.Row
        if approval_status:
            cur = await conn.execute(
                """SELECT id, post_date, day_of_week, message_text, approval_status, created_at, sent_at
                   FROM posts WHERE approval_status = ? ORDER BY post_date DESC LIMIT ?""",
                (approval_status, limit),
            )
        else:
            cur = await conn.execute(
                """SELECT id, post_date, day_of_week, message_text, approval_status, created_at, sent_at
                   FROM posts ORDER BY post_date DESC LIMIT ?""",
                (limit,),
            )
        rows = await cur.fetchall()
        return [dict(r) for r in rows]


async def stats_by_approval(db_path: str) -> dict:
    """Количество постов по каждому статусу — для аналитики."""
    async with await get_connection(db_path) as conn:
        cur = await conn.execute(
            "SELECT approval_status, COUNT(*) AS cnt FROM posts GROUP BY approval_status"
        )
        rows = await cur.fetchall()
        return {row[0]: row[1] for row in rows}
