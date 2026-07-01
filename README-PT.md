# retrokit

Ferramentas de linha de comando para converter imagens de fita cassete de computadores retrô (CAS) para áudio (WAV) e vice-versa, construídas sobre uma arquitetura extensível por plataforma.

## Destaques

- Converta imagens CAS para arquivos de áudio WAV para reprodução em computadores retrô reais
- Decodifique gravações WAV de volta para imagens CAS para arquivamento e emulação
- Inspecione arquivos CAS, CMD e WAV para exibir estrutura detalhada e metadados
- Envolva binários brutos em CAS e extraia binários de volta do CAS
- Interface `Machine` plugável: cada plataforma encapsula sua modulação, layout de blocos e tipos de arquivo
- Suporta atualmente **MSX** e **TRS-80**; projetada para crescer para ZX81, Apple II, Commodore e outros

## Pré-requisitos

- **Go 1.26+** — necessário para compilar a partir do código fonte

## Instalação

### Compilar a partir do Código Fonte

```bash
git clone https://github.com/carlosrabelo/retrokit.git
cd retrokit
make build
```

Instale em `~/.local/bin` (sem necessidade de root):

```bash
make install
```

## Uso

A máquina é sempre obrigatória como prefixo para todos os comandos; apenas `machines` e `help` são de nível superior.

```
retrokit <máquina> <comando> [opções] <entrada> <saída>
retrokit <máquina> info <arquivo>
retrokit machines
```

### cas2wav — Converter CAS para WAV

```bash
./bin/retrokit msx cas2wav -b 1200 programa.cas programa.wav
./bin/retrokit trs80 cas2wav programa.cas programa.wav
```

### wav2cas — Converter WAV para CAS

```bash
./bin/retrokit msx wav2cas -t 5 gravacao.wav saida.cas
```

### bin2cas — Envolver um binário em CAS

```bash
./bin/retrokit msx bin2cas -t bin --addr C000 --exec C000 programa.bin programa.cas
```

### cas2bin — Extrair um binário do CAS

```bash
./bin/retrokit msx cas2bin programa.cas programa.bin
```

### info — Inspecionar um arquivo

```bash
./bin/retrokit msx info programa.cas
./bin/retrokit trs80 info programa.cmd
./bin/retrokit msx info gravacao.wav
```

### machines — Listar máquinas suportadas

```bash
./bin/retrokit machines
```

## Máquinas Suportadas

| Máquina | Formato de fita | Taxas (baud) | Tipos de arquivo |
| --- | --- | --- | --- |
| MSX | Kansas City (header de 8 bytes + marcador de tipo) | 1200, 2400 | ASCII, BIN, BASIC |
| TRS-80 | Preâmbulo de 256 bytes + sync 0xA5 | 500 | data (formato de disco CMD também inspecionável) |

## Estrutura do Projeto

```
cmd/retrokit/              # Ponto de entrada da CLI
internal/
    wav/                   # leitura/escrita/inspeção de WAV (compartilhado)
    machine/               # interface Machine + registro
        msx/               # implementação MSX
        trs80/             # implementação TRS-80 (+ parser CMD)
        all/               # registra todas as máquinas via init()
    cas2wav/               # CAS para WAV (delega à Machine)
    wav2cas/               # WAV para CAS (delega à Machine)
    bin2cas/               # binário para CAS (delega à Machine)
    cas2bin/               # CAS para binário (delega à Machine)
    info/                  # inspeção de arquivos (delega à Machine)
bin/                       # Binários compilados (ignorados pelo git)
.make/                    # Scripts de build e instalação
```

### Adicionando uma nova plataforma

Implemente a interface `machine.Machine` em um novo pacote sob `internal/machine/<nome>/`, registre-o em `init()` e adicione um import em branco em `internal/machine/all/all.go`. Nenhum pacote de comando precisa mudar.

## Desenvolvimento

```bash
make build      # Compilar binário em bin/retrokit
make test       # Executar todos os testes
make fmt        # Formatar código
make lint       # Executar linter
make install    # Instalar em ~/.local/bin
```

## Licença

Este projeto é licenciado sob a Licença MIT — veja [LICENSE](LICENSE) para detalhes.
