# -*- coding: utf-8 -*-
"""
Этап 1: без нейросети. Варианты текста берутся из заготовленного списка.
Для этапа 2 можно заменить на вызов любого провайдера через интерфейс.
"""
import random
from typing import List

from ..interface import TextGenerator

# Заготовки с разной степенью абсурда и тона
TEMPLATES = [
    "сегодня {day} если что",
    "сегодня {day}.",
    "доброе утро, сегодня {day}.",
    "напоминаю: сегодня {day}.",
    "{day} сегодня, да.",
    "вот и {day} настал.",
    "день недели: {day}. всё.",
    "сегодня {day}, живите с этим.",
    "снова {day}. ничего не поделаешь.",
    "если что — сегодня {day}.",
]


class SimpleGenerator(TextGenerator):
    """Генератор из фиксированного списка шаблонов."""

    def __init__(self, templates: List[str] | None = None, **kwargs):
        self._templates = templates or TEMPLATES

    async def generate_variants(self, day_name: str, count: int = 5) -> List[str]:
        day_lower = day_name.lower()
        filled = [t.format(day=day_lower) for t in self._templates]
        random.shuffle(filled)
        return filled[:count]
