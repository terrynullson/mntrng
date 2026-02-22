# Thresholds & Rules — HLS Monitoring Platform

## A) Statuses

- OK — поток доступен, свежий, без критических артефактов.
- WARN — деградация качества / пограничные значения.
- FAIL — недоступен / не свежий / критическая ошибка / длительный freeze / black frame.

## B) Atomic Checks

1) Playlist availability (HTTP, валидный m3u8)
2) Segments availability (N последних сегментов)
3) Freshness (EXT-X-PROGRAM-DATE-TIME если есть; иначе прогресс sequence)
4) Declared bitrate (из playlist, если доступно)
5) Effective bitrate (по скачанным сегментам за окно)
6) Freeze (повтор/почти-повтор кадров)
7) Black frame (доля “тёмных” кадров)

## C) Default Thresholds (Baseline)

Network:
- playlist_timeout_ms: 3000
- segment_timeout_ms: 5000
- segments_sample_count: 5
- retries: 2 (exponential backoff)

Freshness:
- freshness_warn_sec: 10
- freshness_fail_sec: 30

If PROGRAM-DATE-TIME отсутствует:
- warn: если sequence не меняется > 3 polling
- fail: если sequence не меняется > 8 polling

Bitrate:
- effective_bitrate_warn_ratio: < 0.75 от declared (окно 5)
- effective_bitrate_fail_ratio: < 0.50 от declared (окно 5)
Если declared неизвестен:
- warn если effective < 500 kbps
- fail если effective < 250 kbps

Freeze:
- freeze_warn_sec: 5
- freeze_fail_sec: 15

Black frame:
- blackframe_warn_ratio: 0.50 (>=50% тёмных кадров) в окне >= 3s
- blackframe_fail_ratio: 0.80 (>=80% тёмных кадров) в окне >= 8s

Concurrency (ENV, defaults):
- max_worker_concurrency_per_company: 2
- max_worker_concurrency_total: 10

## D) Aggregation Rule

- Любой FAIL чек -> общий FAIL
- WARN при отсутствии FAIL, если >= 1 WARN чек
- Иначе OK

## E) Alerts Anti-Spam (Telegram Alerts Bot)

- Отправка при переходах:
  - OK -> WARN
  - WARN -> FAIL
  - FAIL -> OK (recovered)
- streak: FAIL/WARN отправлять только после 2 подряд результатов одного уровня
- cooldown: не слать повтор одного и того же статуса чаще чем раз в 10 минут на stream

## F) Retry / Hard Timeout / Idempotency

- Network retry: до 2 попыток с backoff
- Hard timeout на job: 30 секунд
- Worker идемпотентный по ключу: (stream_id + planned_at)
