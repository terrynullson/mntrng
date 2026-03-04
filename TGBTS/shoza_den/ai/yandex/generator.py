# -*- coding: utf-8 -*-
"""
Яндекс — вся логика работы с API (токены, запросы) живёт в этой папке.
Этап 2: реализовать вызов Yandex GPT и вернуть 5 вариантов.
"""
from typing import List

from ..interface import TextGenerator


class YandexGenerator(TextGenerator):
    """Генератор через Yandex GPT. Пока заглушка."""

    def __init__(self, api_key: str = "", folder_id: str = "", **kwargs):
        self._api_key = api_key
        self._folder_id = folder_id

    async def generate_variants(self, day_name: str, count: int = 5) -> List[str]:
        # TODO: вызов Yandex API с промптом про день недели, парсинг ответа на N вариантов
        raise NotImplementedError(
            "Yandex: добавьте вызов API (см. ai/yandex/) и верните список из 5 вариантов."
        )
