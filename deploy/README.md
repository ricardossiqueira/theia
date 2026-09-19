# Implantação com systemd

`orangepi-monitor.service` executa o monitor como o usuário de sistema sem
login `orangepi-monitor`. A configuração YAML fica em
`/etc/orangepi-monitor/config.yaml` e as credenciais MQTT ficam em
`/etc/orangepi-monitor/environment`, separadas do repositório.

No Orange Pi, depois de atualizar o projeto, instale o binário e a unidade:

```bash
cd ~/orangepi-monitor
just install-service
```

Crie `/etc/orangepi-monitor/environment` com os nomes definidos no YAML:

```ini
ORANGEPI_MONITOR_MQTT_USERNAME=orangepi-monitor
ORANGEPI_MONITOR_MQTT_PASSWORD=the-monitor-mqtt-password
```

Proteja o arquivo e habilite o serviço no boot:

```bash
sudo chown root:orangepi-monitor /etc/orangepi-monitor/environment
sudo chmod 640 /etc/orangepi-monitor/environment
just enable-service
systemctl status orangepi-monitor.service
```

Consulte os logs operacionais com:

```bash
just service-logs
```

## CI e atualizações automáticas

`.github/workflows/ci.yml` executa verificação de formatação, testes, `vet` e
build Linux ARM64 a cada push ou pull request para `main`. O Orange Pi não
aceita conexões de entrada do GitHub: o timer
`orangepi-monitor-update.timer` verifica `main` a cada cinco minutos, executa
testes, `vet`, build e validação offline da configuração instalada antes de
reiniciar o monitor.

O atualizador jamais copia um YAML do Git e não lê nem sobrescreve
`/etc/orangepi-monitor/environment`. Ele aceita apenas um fast-forward. Se o
novo binário ou unidade não iniciar, restaura as versões anteriores.

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
