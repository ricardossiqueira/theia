# Orange Pi Monitor

Serviço Linux independente que coleta métricas do Orange Pi e as publica no Mosquitto local como telemetria MQTT.

Ele é deliberadamente separado do `iot-gateway`: o monitor só coleta e publica; o gateway valida, enfileira e, futuramente, roteia os dados para consumidores como o CYD.

```text
orangepi-monitor ── telemetry ──> Mosquitto ──> iot-gateway
                                              └──> CYD, via rota futura
```

## Contrato MQTT

Com `device_id: orangepi-monitor`, o serviço publica em:

```text
devices/orangepi-monitor/telemetry
```

As mensagens usam QoS 1, não são retained e incluem `message_id` UUID v4 e `timestamp` UTC. Consulte [config.example.yaml](configs/config.example.yaml) e a unidade [systemd](deploy/orangepi-monitor.service).

## Desenvolvimento

```bash
go test ./...
go build -o bin/orangepi-monitor ./cmd/orangepi-monitor
```

O coletor real requer Linux e lê somente interfaces locais padrão: `/proc`, `/sys/class/thermal` e o sistema de arquivos raiz.
