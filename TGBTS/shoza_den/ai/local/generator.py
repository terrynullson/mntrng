# -*- coding: utf-8 -*-
"""
Локальная модель (Ollama, LM Studio и т.д.) — отдельный модуль.
Этап 2: реализовать запрос к localhost и парсинг ответа.
"""
from typing import List

from ..interface import TextGenerator


class LocalGenerator(TextGenerator):
    """Генератор через локальный endpoint. Пока заглушка."""

    def __init__(self, base_url: str = "http://localhost:11434", model: str = "", **kwargs):
        self._base_url = base_url
        self._model = model

    async def generate_variants(self, day_name: str, count: int = 5) -> List[str]:
        # TODO: запрос к Ollama/LM Studio, промпт, парсинг 5 вариантов
        raise NotImplementedError(
            "Local: добавьте вызов локального API (см. ai/local/) и верните список из 5 вариантов."
        )
