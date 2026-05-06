# Документация OLCRTC

Основные документы:

- [fast.md](fast.md) - быстрый старт.
- [manual.md](manual.md) - ручная сборка и запуск.
- [settings.md](settings.md) - матрица carrier/transport и флаги CLI.
- [LIFETIME_FEATURE.md](LIFETIME_FEATURE.md) - автоматическая ротация комнат через `-lifetime`.
- [USAGE_EXAMPLES.md](USAGE_EXAMPLES.md) - примеры запуска ротации комнат.
- [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md) - техническое описание изменений.

## Ротация комнат

Серверный флаг `-lifetime <секунды>` включает автоматическую смену комнаты. При `0` или без флага ротация выключена.

```sh
./olcrtc -mode srv \
  -link direct \
  -transport datachannel \
  -carrier jazz \
  -id any \
  -key <hex-key> \
  -dns 1.1.1.1:53 \
  -data data \
  -lifetime 300
```

Клиент запускается как обычно. При служебном сообщении `room_rotate` он закрывает текущий link и делает до 3 попыток переподключения в новую комнату с timeout 30 секунд на попытку.

Во время ротации уже открытые TCP-соединения через SOCKS могут оборваться. Новые соединения пойдут через новую комнату после переподключения.
