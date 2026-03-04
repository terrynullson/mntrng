# Запуск одного поста на сегодня (для Планировщика заданий Windows).
# Действие: Program = py или python, Arguments = run.py once, Start in = папка telegram-day-bot.
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot
& py -3 run.py once
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
