package com.futhub.payment.outbox;

import java.time.Instant;
import java.util.UUID;

public record OutboxRow(UUID id, UUID correlationId, String type, String routingKey, String payload, Instant createdAt) {
}
