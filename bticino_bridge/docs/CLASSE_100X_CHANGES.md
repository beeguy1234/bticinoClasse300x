# Wijzigingen voor BTicino Classe 100X compatibiliteit

## Overzicht
Dit document beschrijft de aanpassingen die gemaakt zijn om de `bticino_bridge` (oorspronkelijk ontworpen voor de Classe 300X) werkend te krijgen op een **BTicino Classe 100X** (C100X16E).

## Gewijzigde bestanden (3 stuks)

### 1. `configs/config.yaml` — Configuratie aanpassingen
- **MQTT**: IP/credentials aangepast naar eigen Raspberry Pi (`192.168.0.107`)
- **SIP transport**: `tcp` → `udp` (Flexisip op de C100X luistert op UDP, niet TCP)
- **SIP domain**: Aangepast naar het unieke apparaat-domein van de C100X (`4b7fc631-...`)
- **SIP username/password**: `user1` / `12345` (uit `/etc/flexisip/default_users.txt`)
- **HomeKit**: Uitgeschakeld (niet nodig)
- **Streaming**: Uitgeschakeld (C100X ondersteunt geen *7*300 RTP-duplicatie)
- **Additional locks**: 3 deuren geconfigureerd (Blok G, Hollebeek, Steenweg)
- **Log level**: `debug` → `info`

### 2. `pkg/sip/snapshot.go` — Framebuffer snapshot (HOOFDWIJZIGING)
**Probleem**: De C300X-methode (`*7*300` → GStreamer H.264-naar-JPEG) werkt niet op de C100X. De intercom ACKt het commando maar stuurt geen decodeerbare RTP-pakketten naar lokale UDP-poorten.

**Oplossing**: In plaats van GStreamer/RTP wordt het **Linux framebuffer** (`/dev/fb0`) rechtstreeks uitgelezen. Wanneer de camera actief is (deurbel, "oogje"-knop), bevat de framebuffer het live camerabeeld. De code:
1. Leest 800×480×4 bytes BGRA uit `/dev/fb0`
2. Converteert BGRA → RGBA met Go's `image` package
3. Encodeert als JPEG met Go's `image/jpeg` — geen externe tools nodig

### 3. `pkg/sip/avmedia_capture.go` — GStreamer pipeline fix
- `videoconvert` element toegevoegd tussen `imxvpudec` en `jpegenc` voor correcte kleurruimte-conversie op de i.MX6 VPU. (Deze fix is relevant als de `*7*300`-methode in de toekomst alsnog werkend wordt gemaakt.)

## Wat werkt op de Classe 100X
- ✅ Deurbesturing via MQTT (3 sloten: Blok G, Hollebeek, Steenweg)
- ✅ Deurbel-notificaties (onderscheid straat vs. etage) via MQTT
- ✅ SIP-registratie (via UDP, direct succesvol)
- ✅ Snapshot via framebuffer (`/api/snapshot`)
- ✅ Web Dashboard
- ✅ Volledige OpenWebNet bus monitoring
- ✅ LED/GPIO status monitoring

## Wat NIET werkt op de Classe 100X
- ❌ RTSP streaming (C100X heeft geen lokale UDP video-duplicatie)
- ❌ HomeKit camera integratie
- ❌ `*7*300` RTP video capture
