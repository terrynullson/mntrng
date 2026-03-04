# -*- coding: utf-8 -*-
"""
Инициализация БД: создание таблиц из schema.sql.
"""
from pathlib import Path
from typing import Optional


def init_db(db_path: str, schema_path: Optional[Path] = None) -> None:
    import sqlite3
    if schema_path is None:
        schema_path = Path(__file__).parent / "schema.sql"
    with open(schema_path, "r", encoding="utf-8") as f:
        sql = f.read()
    conn = sqlite3.connect(db_path)
    conn.executescript(sql)
    conn.commit()
    conn.close()
