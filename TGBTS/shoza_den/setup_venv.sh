#!/bin/sh
# Один раз: создать venv и установить зависимости (для серверов с externally-managed-environment).
set -e
cd "$(dirname "$0")"
python3 -m venv .venv
.venv/bin/pip install -r requirements.txt
echo "Готово. Запуск: .venv/bin/python run.py"
