# Como funciona o serviço de notificações (FutHub)

## A ideia em uma frase
Quando o admin registra algo numa partida (gol, cartão, fim de jogo), o **main** avisa o **notification** por uma fila. O notification guarda o estado da partida e entrega ao vivo para quem é **premium**; quem é **free** só vê um resumo que atualiza de tempos em tempos.

## Arquitetura (o caminho de um gol)

```
Admin --> main (8080) --> grava no banco + "outbox"
                              |
                              v  (publica a cada 500ms)
                         RabbitMQ  (exchange futhub.events)
                              |
                              v  fila notification.q
                    notification (8082)
                      |-- Processor: valida ordem e aplica no estado
                      |-- Redis: guarda estado/resumo da partida
                      |-- Hub: distribui aos WebSockets conectados
                              |
                              v
                  Usuário premium (WebSocket /stream)
                  Usuário free   (GET /summary)
```

Os passos:
1. O `main` salva o gol e, **na mesma transação**, grava um evento numa tabela *outbox*. Assim, ou os dois acontecem ou nenhum. Um job publica o outbox no RabbitMQ, então nenhum evento se perde se o RabbitMQ estiver fora do ar.
2. O RabbitMQ entrega na fila `notification.q`.
3. O `notification` consome, aplica no estado (guardado no Redis) e empurra para quem está conectado.

## Regras de negócio

### Quem vê o quê

| Usuário | `/stream` (tempo real) | `/summary` |
|---|---|---|
| Premium | Sim, a cada evento | Sim |
| Free | **403 `premium_required`** | Sim, atualizado só a cada N segundos |
| Sem login | 401 | 401 |

- **Premium** é: admin (`ADMIN_IS_PREMIUM=true`), ou um ID listado em `PREMIUM_USER_IDS`, ou um usuário ativo na tabela `premium_users`. Hoje as duas primeiras são atalhos de desenvolvimento, até a saga de assinatura/pagamento alimentar essa tabela.
- **Resumo do free:** um job (`SnapshotJob`) copia o estado para o resumo a cada `SNAPSHOT_INTERVAL_SECONDS` (300s no Docker). Por isso o free via 0x0 mesmo com o gol já registrado. É o diferencial do plano pago.
- **Partida só existe "ao vivo" entre o `start` e o `finish`.** Depois de finalizada, `/stream` e `/summary` retornam `match_not_live` e apontam para o `main` para consultar o histórico.
- **Ao conectar no stream**, o premium recebe primeiro o estado completo (`MatchState`) e depois as atualizações incrementais. Quem entra no meio do jogo não perde nada.
- **Placar:** é a soma dos eventos `GOAL` (calculado no `main`). Se um evento é removido por engano (`MatchEventRemoved`), o placar volta.

## Garantias de mensageria

1. **Sequência por partida:** cada mensagem carrega um `sequence` (1, 2, 3...).
   - `sequence <= último aplicado`: **duplicada**, ignorada (idempotência).
   - `sequence` pulou (ex.: chegou 5 e o último foi 3): erro `ErrSequenceGap`, vai para retry até a mensagem faltante chegar.
   - Mensagem de uma partida que não começou (sem `MatchStarted`) também espera.
2. **Ack manual:** a mensagem só sai da fila depois de processada com sucesso. Se o serviço cair no meio, o RabbitMQ reentrega.
3. **Retry e DLQ:**
   - Erro temporário: vai para `notification.retry`, com contador `x-retry-count`.
   - Estourou `MAX_RETRIES` (3): vai para `notification.dlq` (dead letter queue), com o motivo no header.
   - Mensagem inválida (JSON quebrado, tipo desconhecido): direto para a DLQ, sem retry inútil.
   - A movimentação usa *publisher confirms*: só dá ack na original depois que o broker confirmou a cópia. Se não confirmar, devolve a mensagem à fila.
4. **Reconexão automática** ao RabbitMQ com backoff exponencial (1s até 30s).
5. **Prefetch (5):** o consumidor pega no máximo 5 mensagens por vez, evitando sobrecarga.

## Tempo real (WebSocket)
- Autenticação por JWT na query (`?token=`), porque o navegador não consegue mandar header no WebSocket.
- O **Hub** mantém uma lista de inscritos por partida e publica cada atualização para todos eles.
- Ping/pong detecta conexões mortas. No desligamento, o serviço fecha as conexões com "going away".
- Quando chega `MatchFinished`, a atualização sai marcada como `final` e o stream encerra.

## Um bug que achei testando
O middleware de log embrulhava o `ResponseWriter` e impedia o upgrade para WebSocket (erro 500). Os testes unitários não pegaram porque não passavam por esse middleware. Corrigimos adicionando `Hijack` ao wrapper (`services/notification/internal/httpx/httpx.go`). Lição: **teste ponta a ponta acha o que teste unitário não acha**.

## Roteiro de demo
1. Mostrar `docker compose ps` (tudo saudável).
2. Abrir o `wscat` como admin (premium) numa partida.
3. Registrar gol e cartão, e mostrar chegando ao vivo.
4. Mostrar o usuário free tomando `premium_required` no stream e vendo o resumo defasado.
5. Dar `finish` e mostrar `match_not_live`.
6. Opcional: abrir o painel do RabbitMQ (http://localhost:15672, guest/guest) e mostrar as filas `notification.q`, `.retry` e `.dlq`.

## Referência rápida de comandos

```powershell
docker compose up -d --build        # sobe tudo
docker compose ps                   # status
docker compose logs -f notification
.\scripts\setup-match.ps1           # cria campeonato, times e partida (imprime TOKEN, MATCH, HOME)

# iniciar partida, resumo, stream
Invoke-RestMethod -Method Post "http://localhost:8080/matches/$m/start" -Headers $h
Invoke-RestMethod "http://localhost:8082/matches/$m/summary" -Headers $h
npx wscat -c "ws://localhost:8082/matches/$m/stream?token=$token"
```

Rotas do notification (porta 8082): `GET /matches/{id}/stream` (WebSocket, premium) e `GET /matches/{id}/summary`.
