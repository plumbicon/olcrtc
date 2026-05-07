<div align="center">

<img src="https://github.com/openlibrecommunity/material/blob/master/olcrtc.png" width="250" height="250">

![License](https://img.shields.io/badge/license-WTFPL-0D1117?style=flat-square&logo=open-source-initiative&logoColor=green&labelColor=0D1117)
![Golang](https://img.shields.io/badge/-Golang-0D1117?style=flat-square&logo=go&logoColor=00A7D0)

</div>

# Настройки

## Матрица совместимости

Сначала выбери что с чем работает:

| Transport | telemost | jazz | wbstream |
|-----------|:--------:|:----:|:--------:|
| datachannel | ✗ | ✓ | ✓ |
| vp8channel | ✓ | ✓ | ✓ |
| seichannel | ✗ | ✓ | ✓ |
| videochannel | ✓ | ✓ | ✓ |

Скорость по убыванию: datachannel > vp8channel > seichannel > videochannel

---

## Обязательные флаги

Вместо флагов подключения можно передать JSON через `-config <path>`.
Для сервера JSON может быть массивом локаций. Значения, указанные отдельными
CLI-флагами, перекрывают значения из JSON во всех локациях.

| Флаг | Что вводить |
|------|-------------|
| `-config` | Путь к JSON-конфигу |
| `-mode` | `srv` на сервере, `cnc` на клиенте |
| `-carrier` | `telemost`, `jazz` или `wbstream` |
| `-transport` | `datachannel`, `vp8channel`, `seichannel` или `videochannel` |
| `-id` | Room ID. Для jazz/wbstream можно `any` - сгенерируется автоматически |
| `-client-id` | Общий идентификатор клиента. Должен совпадать на сервере и клиенте |
| `-key` | Ключ шифрования hex 64 символа. Генерация: `openssl rand -hex 32` |
| `-link` | Всегда `direct` |
| `-data` | Всегда `data` |
| `-dns` | DNS-сервер, например `1.1.1.1:53` |

---

## Необязательные флаги

| Флаг | Описание | По умолчанию |
|------|----------|:------------:|
| `-dns` | DNS-сервер | `1.1.1.1:53` |
| `-lifetime` | Время жизни комнаты в секундах. Работает на сервере, `0` = без ротации | `0` |
| `-port` | HTTP-порт, на котором сервер отдаёт клиентскую подписку в формате `sub.md` | `0` |
| `--debug` | Подробные логи соединений | выкл |

---

## Флаги только для сервера (`-mode srv`)

| Флаг | Описание |
|------|----------|
| `-socks-proxy` | Адрес SOCKS5-прокси для исходящего трафика сервера |
| `-socks-proxy-port` | Порт этого прокси |

---

## Флаги только для клиента (`-mode cnc`)

| Флаг | Описание | По умолчанию |
|------|----------|:------------:|
| `-socks-host` | На каком адресе поднять SOCKS5 | `127.0.0.1` |
| `-socks-port` | На каком порту поднять SOCKS5 | `1080` |

---

## vp8channel

**Рекомендуется: `-vp8-fps 60 -vp8-batch 64`** (числа лучше чётные, больший batch = выше скорость)

| Флаг | Описание | По умолчанию |
|------|----------|:------------:|
| `-vp8-fps` | FPS VP8 потока | `25` |
| `-vp8-batch` | Кадров за тик | `1` |

---

## videochannel

**Рекомендуется: `-video-codec qrcode -video-w 1080 -video-h 1080 -video-fps 60 -video-bitrate 5000k -video-hw none`**

| Флаг | Описание | По умолчанию |
|------|----------|:------------:|
| `-video-codec` | `qrcode` или `tile` | `qrcode` |
| `-video-w` | Ширина в пикселях | `1920` |
| `-video-h` | Высота в пикселях | `1080` |
| `-video-fps` | FPS | `30` |
| `-video-bitrate` | Битрейт, например `2M` или `5000k` | `2M` |
| `-video-hw` | Аппаратное ускорение: `none` или `nvenc` | `none` |
| `-video-qr-recovery` | Коррекция ошибок QR: `low` / `medium` / `high` / `highest` | `low` |
| `-video-qr-size` | Размер фрагмента QR в байтах, `0` = авто | `0` |
| `-video-tile-module` | Размер тайла в пикселях 1..270 (только `tile`) | `4` |
| `-video-tile-rs` | Reed-Solomon паритет % 0..200 (только `tile`) | `20` |

Для codec `tile` нужно точно `1080x1080`.

---

## seichannel

**Рекомендуется: `-fps 60 -batch 64 -frag 900 -ack-ms 2000`**

| Флаг | Описание | По умолчанию |
|------|----------|:------------:|
| `-fps` | FPS H264 потока | `60` |
| `-batch` | Кадров за тик | `64` |
| `-frag` | Размер фрагмента в байтах | `900` |
| `-ack-ms` | Таймаут ACK в миллисекундах | `2000` |

---

## datachannel

Дополнительных флагов нет - всё по умолчанию.

---

## JSON-конфиг

Основной формат конфига - объект версии 4 со списком локаций:

```json
{
  "version": 4,
  "active_location_id": "netherlands",
  "name": "ScumVPN",
  "port": 8080,
  "color": "#4A90E2",
  "icon": "vpn",
  "used": "10mb/10gb",
  "available": "9.99gb",
  "locations": [
    {
      "storage_id": "netherlands",
      "name": "Netherlands",
      "color": "#4A90E2",
      "icon": "nl",
      "mimo": "NL / olcng free sub / IPv6",
      "client-id": "client-1",
      "endpoint": {
        "room_id": "019df548-e1ab-7413-8199-2b833e90041c",
        "key": "e830d36f7be8cfb04a741fc1a5e2ddf8ff04f30985dc070616483f939ad5fafe"
      },
      "carrier": "wbstream",
      "transport": {
        "type": "datachannel"
      },
      "lifetime": 600
    },
    {
      "storage_id": "telemost-vp8",
      "name": "Telemost VP8",
      "client-id": "client-1",
      "endpoint": {
        "room_id": "room-2",
        "key": "e830d36f7be8cfb04a741fc1a5e2ddf8ff04f30985dc070616483f939ad5fafe"
      },
      "carrier": "telemost",
      "transport": {
        "type": "vp8channel",
        "vp8": {
          "fps": 60,
          "batch": 64
        }
      }
    }
  ]
}
```

`active_location_id` используется только клиентом (`mode: cnc`): клиент
выбирает локацию, у которой `storage_id` совпадает с этим значением. Сервер
игнорирует `active_location_id` и поднимает все локации.

Верхнеуровневые поля `name`, `color`, `icon`, `used`, `available`, `refresh`
нужны для подписки. Они попадут в HTTP-ответ как глобальные строки `#...`.

Поля локации `name`, `color`, `icon`, `used`, `available`, `ip`, `comment`,
`mimo` тоже нужны для подписки. Они попадут в локальные строки `##...`, а
`mimo` попадёт в конец URI после `$`.

`port` на верхнем уровне включает HTTP-раздачу подписки. То же значение можно
передать флагом `-port`; флаг перекрывает JSON.

Запуск:

```sh
./olcrtc -mode cnc -link direct -data data -dns 1.1.1.1:53 -config client.json
```

Для сервера та же структура поднимает отдельное подключение к каждой комнате и
обслуживает клиентов во всех локациях:

```sh
./olcrtc -mode srv -link direct -data data -dns 1.1.1.1:53 -config server.json
```

Если задан `port`, сервер дополнительно начинает принимать HTTP GET-запросы и
отдаёт подписку в формате [sub.md](sub.md):

```text
#name: ScumVPN
#update: 1778012000
#refresh: 1778012600

olcrtc://wbstream?datachannel@019e033e-...#e830...fafe%client-1$NL / olcng free sub / IPv6
##name: Netherlands
##color: #4A90E2
##icon: nl
```

`#update` сервер пишет автоматически как Unix time последнего обновления данных
подписки. Значение обновляется при старте и при пересоздании комнаты. Если у
любой локации задан `lifetime`, `#refresh` будет Unix time следующего
обновления: `update + min(lifetime)`. При разных `lifetime` используется
минимальный положительный, чтобы клиент успел обновиться до ближайшей ротации.

Если `lifetime` нигде не задан, сервер отдаёт явно заданный верхнеуровневый
`refresh`, если он есть.

Для обратной совместимости по-прежнему читаются старые форматы: одиночный
объект для одной локации и массив локаций.

```json
[
  {
    "label": "wbstream-vp8",
    "endpoint": {
      "room_id": "any",
      "key": "e830d36f7be8cfb04a741fc1a5e2ddf8ff04f30985dc070616483f939ad5fafe"
    },
    "carrier": "wbstream",
    "transport": {
      "type": "vp8channel",
      "vp8": {
        "fps": 60,
        "batch": 64
      }
    }
  },
  {
    "label": "telemost-vp8",
    "endpoint": {
      "room_id": "room-2",
      "key": "e830d36f7be8cfb04a741fc1a5e2ddf8ff04f30985dc070616483f939ad5fafe"
    },
    "carrier": "telemost",
    "transport": {
      "type": "vp8channel",
      "vp8": {
        "fps": 60,
        "batch": 64
      }
    }
  }
]
```

`label` необязателен, но полезен в логах. Если WBStream или Jazz создают
комнату из `room_id: "any"`, id комнаты будет выведен вместе с label:

Для Telemost `room_id: "any"` не поддерживается: создай встречу на
telemost.yandex.ru и передай ID из ссылки.

```text
WB Stream room created [wbstream-vp8]: 019e000b-...
To connect client use [wbstream-vp8]: -id 019e000b-...
```

Общие для всех локаций параметры удобно передать флагами:

```sh
./olcrtc -mode srv -link direct -data data -dns 1.1.1.1:53 -config server.json
```

Если указывать их в JSON, то `data` должен быть одинаковым во всех локациях,
потому что файлы имен загружаются один раз на процесс.

Для сервера в каждой локации при необходимости можно указать:

```json
{
  "server": {
    "socks_proxy": "127.0.0.1",
    "socks_proxy_port": 9050
  },
  "lifetime": 3600
}
```

Для `videochannel` параметры можно положить в `"video"` или в
`"transport.video"`:

```json
{
  "transport": {
    "type": "videochannel"
  },
  "video": {
    "codec": "qrcode",
    "width": 1080,
    "height": 1080,
    "fps": 60,
    "bitrate": "5000k",
    "hw": "none"
  }
}
```

---

## Готовые команды

### telemost + seichannel

```sh
# сервер
./olcrtc -mode srv -carrier telemost -transport seichannel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -fps 60 -batch 64 -frag 900 -ack-ms 2000

# клиент
./olcrtc -mode cnc -carrier telemost -transport seichannel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -socks-host 127.0.0.1 -socks-port 1080 \
  -fps 60 -batch 64 -frag 900 -ack-ms 2000
```

### telemost + vp8channel

```sh
# сервер
./olcrtc -mode srv -carrier telemost -transport vp8channel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -vp8-fps 60 -vp8-batch 64

# клиент
./olcrtc -mode cnc -carrier telemost -transport vp8channel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -socks-host 127.0.0.1 -socks-port 1080 \
  -vp8-fps 60 -vp8-batch 64
```

### jazz + datachannel (максимальная скорость)

```sh
# сервер - room ID создастся сам, смотри логи
./olcrtc -mode srv -carrier jazz -transport datachannel \
  -id any -client-id <client-id> -key <hex-key> -link direct -data data

# клиент
./olcrtc -mode cnc -carrier jazz -transport datachannel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -socks-host 127.0.0.1 -socks-port 1080
```

### telemost + videochannel (крайний случай)

```sh
# сервер
./olcrtc -mode srv -carrier telemost -transport videochannel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -video-codec qrcode -video-w 1080 -video-h 1080 \
  -video-fps 60 -video-bitrate 5000k -video-hw none

# клиент
./olcrtc -mode cnc -carrier telemost -transport videochannel \
  -id <room-id> -client-id <client-id> -key <hex-key> -link direct -data data \
  -socks-host 127.0.0.1 -socks-port 1080 \
  -video-codec qrcode -video-w 1080 -video-h 1080 \
  -video-fps 60 -video-bitrate 5000k -video-hw none
```
