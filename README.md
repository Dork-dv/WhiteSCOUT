# WhiteScout

WhiteScout é um **crawler de endpoints web** desenvolvido em Go, focado em coletar URLs interessantes de um site alvo. Ele suporta diferentes níveis de filtragem e profundidade, e pode validar se os endpoints estão ativos.

<br>

## Funcionalidades

- Coleta endpoints de páginas web e arquivos JS/HTML.
- Filtragem por relevância:
  - **Rasa**: apenas endpoints interessantes (admin, login, API, dashboard etc.).
  - **Padrão**: ignora arquivos estáticos (CSS, JS, imagens).
  - **Profunda**: coleta todos os endpoints possíveis.
- Suporta múltiplas threads.
- Delay configurável entre requisições.
- Valida se os endpoints estão ativos (HTTP HEAD/GET).
- Exporta resultados para `endpoints.txt`.
- Mostra progresso ao vivo com RPS (requests por segundo).

## Pré-requisitos

- Go 1.20+ instalado
- Sistema operacional compatível (Linux, macOS, Windows)

## Instalação

1. Clone o repositório:

```bash
git clone https://github.com/seuusuario/WhiteScout.git
cd WhiteScout
./WhiteScout
