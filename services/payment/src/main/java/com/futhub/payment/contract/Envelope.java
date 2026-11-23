package com.futhub.payment.contract;

import java.time.Instant;
import java.util.UUID;

import com.fasterxml.jackson.databind.JsonNode;

public record Envelope(
        UUID messageId,
        UUID correlationId,
        String type,
        int version,
        Instant occurredAt,
        JsonNode payload) {
}
