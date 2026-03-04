# -*- coding: utf-8 -*-
"""
Фабрика провайдеров. По настройке AI_PROVIDER возвращает нужную реализацию.
Добавление нового провайдера = новая папка + одна строка здесь.
"""
from typing import Any, Optional

from .interface import TextGenerator
from .simple.generator import SimpleGenerator


def get_generator(provider: str, config: Optional[Any] = None, **kwargs) -> TextGenerator:
    """
    :param provider: simple | yandex | deepseek | local
    :param config: объект config с атрибутами (YANDEX_API_KEY, DEEPSEEK_API_KEY и т.д.);
                  если передан, провайдеру передаются только нужные ему поля
    :param kwargs: доп. аргументы (переопределяют config)
    """
    provider = (provider or "simple").strip().lower()
    if config is not None:
        if provider == "simple":
            return SimpleGenerator(**kwargs)
        if provider == "yandex":
            from .yandex.generator import YandexGenerator
            return YandexGenerator(
                api_key=getattr(config, "YANDEX_API_KEY", "") or kwargs.get("api_key", ""),
                folder_id=getattr(config, "YANDEX_FOLDER_ID", "") or kwargs.get("folder_id", ""),
            )
        if provider == "deepseek":
            from .deepseek.generator import DeepSeekGenerator
            return DeepSeekGenerator(
                api_key=getattr(config, "DEEPSEEK_API_KEY", "") or kwargs.get("api_key", ""),
                model=kwargs.get("model", "deepseek-chat"),
            )
        if provider == "local":
            from .local.generator import LocalGenerator
            return LocalGenerator(
                base_url=getattr(config, "LOCAL_AI_BASE_URL", "http://localhost:11434") or kwargs.get("base_url", "http://localhost:11434"),
                model=getattr(config, "LOCAL_AI_MODEL", "") or kwargs.get("model", ""),
            )
    else:
        if provider == "simple":
            return SimpleGenerator(**kwargs)
        if provider == "yandex":
            from .yandex.generator import YandexGenerator
            return YandexGenerator(**kwargs)
        if provider == "deepseek":
            from .deepseek.generator import DeepSeekGenerator
            return DeepSeekGenerator(**kwargs)
        if provider == "local":
            from .local.generator import LocalGenerator
            return LocalGenerator(**kwargs)
    raise ValueError(f"Unknown AI provider: {provider}. Use: simple, yandex, deepseek, local")
