# Implantação: container (Podman)

`orangepi-monitor` roda como container Podman (Quadlet -
`orangepi-monitor.container` neste diretório), não mais como processo
solto no host. A imagem é construída na CI hospedada
(`.github/workflows/ci.yml`'s `publish-image` job, a partir do
`Containerfile` na raiz do repositório) e publicada em
`ghcr.io/ricardossiqueira/orangepi-monitor` - nada é compilado no Orange
Pi. A configuração YAML continua em `/etc/orangepi-monitor/config.yaml`;
identidade e credenciais MQTT provisionadas continuam em
`/var/lib/orangepi-monitor` (`identity.json`, `provisioning.json`),
gravadas pelo próprio processo — nenhum segredo é criado manualmente.

O bootstrap completo (instalar Podman, criar os diretórios/ownership,
instalar a unit, habilitar o timer de auto-update) está em
`orangepi-deploy/README.md` (repositório irmão) - comece por lá.

```bash
sudo systemctl enable --now orangepi-monitor.service
systemctl status orangepi-monitor.service
```

## Registro como device v2

Na primeira vez (sem `provisioning.json` em `/var/lib/orangepi-monitor`), o
serviço sobe sem credenciais MQTT e entra numa janela de pareamento
(`device.pairing_window`, padrão 10 minutos): ele se anuncia por mDNS
(`_iot-device._tcp`) e serve `GET /v1/device-info` na porta
`device.http_port`. Dentro dessa janela:

1. Abra `gateway-web` → Discovery e confirme que `orangepi-monitor` aparece.
2. Registre com um clique — o gateway pareia, deriva as credenciais MQTT e
   provisiona o device.
3. O serviço detecta o provisionamento (sem precisar reiniciar), conecta ao
   Mosquitto com a credencial recebida e começa a publicar `telemetry`.

Se a janela expirar antes do registro, reinicie o serviço
(`sudo systemctl restart orangepi-monitor.service`) para abrir uma nova
janela. Depois de provisionado, `/v1/pair` recusa qualquer nova tentativa —
para apagar o provisionamento e reabrir o pareamento de propósito:

```bash
podman run --rm -v /var/lib/orangepi-monitor:/var/lib/orangepi-monitor \
  --user 65532:65532 ghcr.io/ricardossiqueira/orangepi-monitor:latest \
  forget --config /etc/orangepi-monitor/config.yaml
sudo systemctl restart orangepi-monitor.service
```

Consulte os logs operacionais com:

```bash
journalctl -u orangepi-monitor.service -f
```

A unit usa `Notify=true` (Quadlet) para propagar o próprio `sd_notify` do
binário (`internal/sdnotify`) até o systemd do host - `READY=1` só é
sinalizado depois de conectar ao MQTT **e** publicar a primeira telemetria
com sucesso, então `systemctl is-active` depois de um restart é uma
verificação funcional real, não só "o processo não morreu imediatamente".
