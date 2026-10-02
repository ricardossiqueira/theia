# Implantação com systemd

`orangepi-monitor.service` executa o monitor como o usuário de sistema sem
login `orangepi-monitor`. A configuração YAML fica em
`/etc/orangepi-monitor/config.yaml`; identidade e credenciais MQTT
provisionadas ficam em `/var/lib/orangepi-monitor` (`identity.json`,
`provisioning.json`), gravadas pelo próprio processo — nenhum segredo é
criado manualmente.

No Orange Pi, depois de atualizar o projeto, instale o binário e a unidade:

```bash
cd ~/orangepi-monitor
just install-service
just enable-service
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
use `just forget` para apagar o provisionamento e reabrir o pareamento de
propósito.

Consulte os logs operacionais com:

```bash
just service-logs
```

## CI e atualizações automáticas

`.github/workflows/ci.yml` executa verificação de formatação, testes, `vet` e
build Linux ARM64 a cada push ou pull request para `main`. A dependência
`github.com/ricardossiqueira/athena-go` é pública; a CI não precisa de token
adicional para baixá-la.

O Orange Pi não aceita conexões de entrada do GitHub: o timer
`orangepi-monitor-update.timer` verifica `main` a cada cinco minutos, executa
testes, `vet`, build e validação offline da configuração instalada antes de
reiniciar o monitor.

O atualizador jamais copia um YAML do Git, nem sobrescreve
`/etc/orangepi-monitor/config.yaml` ou o conteúdo de
`/var/lib/orangepi-monitor` (identidade e provisionamento). Ele aceita apenas
um fast-forward. Se o novo binário ou unidade não iniciar, restaura as
versões anteriores.

### Acesso Git somente leitura

O atualizador executa Git como `orangepi`; configure uma deploy key de leitura
para este repositório enquanto estiver logado como esse usuário:

```bash
mkdir -p ~/.ssh
chmod 700 ~/.ssh
ssh-keygen -t ed25519 -f ~/.ssh/orangepi-monitor-deploy -C orangepi-monitor-deploy
cat ~/.ssh/orangepi-monitor-deploy.pub
```

Adicione a chave pública em GitHub, em **Settings** → **Deploy keys** do
repositório `orangepi-monitor`, deixando o acesso de escrita desativado. Em
seguida, crie `~/.ssh/config` com:

```text
Host github.com-orangepi-monitor
    HostName github.com
    User git
    IdentityFile ~/.ssh/orangepi-monitor-deploy
    IdentitiesOnly yes
```

Proteja a configuração e use o host para o clone existente:

```bash
chmod 600 ~/.ssh/config ~/.ssh/orangepi-monitor-deploy
ssh -T git@github.com-orangepi-monitor
cd ~/orangepi-monitor
git remote set-url origin git@github.com-orangepi-monitor:ricardossiqueira/orangepi-monitor.git
git fetch origin main
```

### Habilitar o agente de atualização

Depois de atualizar o repositório para uma versão que contenha estes arquivos:

```bash
cd ~/orangepi-monitor
just install-update-agent
just enable-update-agent
sudo systemctl start orangepi-monitor-update.service
just update-agent-status
```

A primeira execução registra a revisão instalada. As seguintes implantam
apenas uma nova revisão de `main`, inclusive atualizações do próprio agente.
Use `just update-agent-logs` para investigar uma falha. Build inválido, YAML
inválido, worktree sujo ou falha de reinício preservam o monitor instalado.

O repositório fica gravável para `orangepi` porque Git e compilação executam
sob esse usuário, mas a instalação final é feita por root. Considere acesso de
escrita ao repositório, à deploy key e pushes diretos para `main` como controle
do Orange Pi. Proteja `main` com o workflow de CI antes de habilitar o agente.
