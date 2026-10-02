# Orange Pi Monitor

Serviço Linux independente que coleta métricas do Orange Pi e as publica no Mosquitto local como telemetria MQTT.

Ele é deliberadamente separado do `iot-gateway`: o monitor só coleta e publica; o gateway valida, enfileira e, futuramente, roteia os dados para consumidores como o CYD.

```text
orangepi-monitor ── telemetry ──> Mosquitto ──> iot-gateway
                                              └──> CYD, via rota futura
```

## Device platform v2

O monitor é um device v2 de verdade (ver
[`iot-device-core-go`](../iot-device-core-go) e
[`DEVICE_PLATFORM_V2_IMPLEMENTATION.md`](../DEVICE_PLATFORM_V2_IMPLEMENTATION.md)):
sem credencial MQTT fixa em lugar nenhum. Na primeira execução, ele gera uma
identidade (`device_uid` + chave Ed25519), se anuncia por mDNS
(`_iot-device._tcp`) e serve `GET /v1/device-info`/`POST /v1/pair`/
`POST /v1/provision` numa janela de pareamento — o operador registra pelo
`gateway-web` (Discovery), que deriva e entrega a credencial MQTT. Só depois
disso o monitor publica em `devices/<device_id>/telemetry` (QoS 1, não
retained, `message_id`/`timestamp` no envelope). Detalhes operacionais e o
schema do manifest estão em [deploy/README.md](deploy/README.md) e
[configs/config.example.yaml](configs/config.example.yaml).

## Desenvolvimento

```bash
go test ./...
go build -o bin/orangepi-monitor ./cmd/orangepi-monitor
```

O coletor real requer Linux e lê somente interfaces locais padrão: `/proc`, `/sys/class/thermal` e o sistema de arquivos raiz.
