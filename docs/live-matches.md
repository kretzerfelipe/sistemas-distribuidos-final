# Live matches: messages published by the main service

This document is for whoever builds the live service (notification). The main service only publishes; everything the live view needs travels in the messages, because the live service cannot read the main database.

## Where the messages go
- Exchange `futhub.events` (topic, durable). Routing keys: `match.started`, `match.event-registered`, `match.event-removed`, `match.finished`.
- The queue `notification.q` is bound to `match.*` (see `infra/rabbitmq/definitions.json`), with `notification.retry` (5 second delay, goes back to `notification.q`) and `notification.dlq`.
- Every message is a JSON envelope: `messageId`, `correlationId`, `type`, `version`, `occurredAt`, `payload`. The schemas are in `contracts/`.

## The four messages
- `MatchStarted`: the match entered the live view. It carries everything the view needs to show the match: championship, home and away team (id and name), scheduled and start time, and the score (0 x 0). `sequence` is 1.
- `MatchEventRegistered`: one play (goal, card, shot, save, foul, substitution, corner, offside). It carries the play, with the player names, and the **score after it**.
- `MatchEventRemoved`: an admin removed a play registered by mistake. It carries the `eventId` and the corrected score.
- `MatchFinished`: the match is over, with the final score.

Players are optional in a play (`player` and `secondaryPlayer` can be `null`). A substitution has both: `player` enters and `secondaryPlayer` leaves.

## What the live service should do (suggested behaviour)
1. **Idempotency (R1):** save the `messageId` already processed, in the same transaction as the effect. A repeated message must change nothing (F3).
2. **Order and gaps:** every message of a match has a `sequence` that grows by one (1, 2, 3, ...), counting every change of the match (start, each play, each removal, finish).
   - `sequence` already applied or lower: duplicate, ignore it.
   - `sequence` bigger than the last applied plus one: a message is missing. Send it to `notification.retry` so it comes back in 5 seconds, after the missing ones arrived. After the maximum number of tries, send it to `notification.dlq`.
3. **Cache of a live match (Redis):** `MatchStarted` creates it (header, score, empty list of plays). `MatchEventRegistered` appends the play and replaces the score. `MatchEventRemoved` removes the play and replaces the score. Keep the last applied `sequence` together.
4. **Someone opens the live view in the middle of the match:** subscribe to the stream first, then read the cache, and drop repeated plays by `sequence`.
5. **`MatchFinished`:** push the last update to the viewers and **delete the cache of the match**. After that, the score and the plays are read from the main service: `GET /matches/{id}` and `GET /matches/{id}/events`.
6. **Premium and free:** the difference is decided on the server. Premium users receive each message at once. Free users read a summary refreshed every `SNAPSHOT_INTERVAL_SECONDS`.

## Guarantees given by the main service
- A play, the match score, its `sequence` and the message are saved in the **same database transaction** (outbox). No play exists without its message, and no message without its play.
- If RabbitMQ is down, messages stay stored and are published in order when it comes back. A message can be delivered more than once (at-least-once), never lost.
- Messages of the same match are published in `sequence` order.
