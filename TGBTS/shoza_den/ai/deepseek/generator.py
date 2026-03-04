# -*- coding: utf-8 -*-
"""
DeepSeek — отдельная реализация. Вся логика (токен, модель, запросы) только здесь.
Этап 2: реализовать вызов API и вернуть 5 вариантов.
"""
from typing import List

from ..interface import TextGenerator


class DeepSeekGenerator(TextGenerator):
    """Генератор через DeepSeek API. Пока заглушка."""

    def __init__(self, api_key: str = "", model: str = "deepseek-chat", **kwargs):
        self._api_key = api_key
        self._model = model

    async def generate_variants(self, day_name: str, count: int = 5) -> List[str]:
        # TODO: вызов DeepSeek API, промпт про день недели, парсинг 5 вариантов
        raise NotImplementedError(
            "DeepSeek: добавьте вызов API (см. ai/deepseek/) и верните список из 5 вариантов."
        )
