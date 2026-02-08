![Banner](.github/banner.png)

## Overview

ChiguiCifras es un bot de Telegram que entrega tasas de cambio en tiempo real para VES (Bolívar venezolano), consumiendo
la API de [**fxrates**](github.com/sig-0/fxrates).

## Comandos

| Comando                                    | Descripción                         |
|--------------------------------------------|-------------------------------------|
| `/start`                                   | Mensaje de bienvenida               |
| `/tasa`                                    | Tasas del día (USD, EUR, USDT)      |
| `/tasa USD`                                | Tasa de un par específico vs VES    |
| `/tasa USD COP`                            | Par explícito                       |
| `/dolar`                                   | Atajo USD/VES                       |
| `/euro`                                    | Atajo EUR/VES                       |
| `/usdt`                                    | Atajo USDT/VES (Compra/Venta)       |
| `/monedas`                                 | Listar monedas disponibles          |
| `/ayuda`                                   | Ayuda                               |
| `/suscribir <horario\|diario>`             | Suscribirse a resúmenes automáticos |
| `/desuscribir`                             | Desuscribirse de resúmenes          |
| `/alerta <moneda> <arriba\|abajo> <valor>` | Crear alerta de precio (máx 5)      |
| `/alertas`                                 | Listar alertas activas              |
| `/borraralerta <id>`                       | Eliminar una alerta por ID          |

Modo inline:

- Usa `@ChiguiCifrasBot USD VES` o `@ChiguiCifrasBot USD` (destino VES por defecto)

## Configuración

La configuración parte de valores por defecto y se puede sobrescribir con un TOML y/o variables de entorno. Si existe un
`.env`, se carga automáticamente.

Variables principales:

- `CHIGUI_TELEGRAM_TOKEN` (requerida)
- `CHIGUI_WEBHOOK_URL` (opcional, **HTTPS** para modo webhook)
- `CHIGUI_WEBHOOK_SECRET_TOKEN` (requerida si usas webhook)
- `CHIGUI_WEBHOOK_LISTEN_ADDR` (opcional, default `0.0.0.0:8080`, solo webhook)
- `CHIGUI_FXRATES_URL` (opcional, default `https://api.ojoporciento.com`)
- `CHIGUI_FXRATES_TIMEOUT` (opcional, default `10s`)
- `CHIGUI_DATABASE_URL` (opcional, PostgreSQL; habilita suscripciones y alertas)
- `CHIGUI_BROADCAST_INTERVAL` (opcional, default `1m`)
- `CHIGUI_ALERT_INTERVAL` (opcional, default `5m`)

Flags disponibles:

- `--config` (ruta a TOML)
- `--listen` (override del listen addr)

Si `CHIGUI_WEBHOOK_URL` no está definida, el bot usa long polling y elimina cualquier webhook previo.

## Build y ejecución

```bash
go build -o build/server ./cmd

CHIGUI_TELEGRAM_TOKEN="..." \
CHIGUI_WEBHOOK_URL="https://tu-dominio.com/telegram/webhook" \
CHIGUI_WEBHOOK_SECRET_TOKEN="..." \

./build/server serve
```

Al iniciar, el bot registra el webhook automáticamente usando `CHIGUI_WEBHOOK_URL`. El servidor local expone:

- El endpoint del webhook en el path de esa URL.
- `GET /health` para health checks.

### Migraciones

Si usas la base de datos (suscripciones y alertas):

```bash
CHIGUI_DATABASE_URL="postgres://user:pass@host:5432/db" \
./build/server sql migrate 001_init.sql
```
