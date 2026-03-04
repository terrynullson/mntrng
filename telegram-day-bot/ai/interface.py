# -*- coding: utf-8 -*-
"""
Единый интерфейс для всех AI-провайдеров.
Основной код бота работает только с этим интерфейсом — смена провайдера
в настройках не ломает логику.
"""
from abc import ABC, abstractmethod
from typing import List


class TextGenerator(ABC):
    """Генератор текстов про день недели. Реализации: Simple, Yandex, DeepSeek, Local."""

    @abstractmethod
    async def generate_variants(self, day_name: str, count: int = 5) -> List[str]:
        """
        Сгенерировать несколько вариантов текста про день недели.

        :param day_name: название дня по-русски (понедельник, вторник, ...)
        :param count: сколько вариантов вернуть
        :return: список непустых строк (до count штук)
        """
        pass

    async def generate_stub(self, day_name: str) -> str:
        """
        Текст «заглушки», когда админ не одобрил вовремя.
        Можно переопределить в провайдере для своего стиля.
        """
        return (
            f"Ну мой господин сегодня мне нихера не одобрил, "
            f"походу занимается херней какой-то снова, поэтому сегодня {day_name}."
        )
