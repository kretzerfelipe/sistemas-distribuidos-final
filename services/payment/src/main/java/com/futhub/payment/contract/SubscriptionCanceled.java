package com.futhub.payment.contract;

import java.util.UUID;

import com.fasterxml.jackson.databind.JsonNode;

public record SubscriptionCanceled(UUID subscriptionId, UUID userId, String reason) {

    public static SubscriptionCanceled from(JsonNode payload) {
        return new SubscriptionCanceled(
                Payloads.uuid(payload, "subscriptionId"),
                Payloads.uuid(payload, "userId"),
                payload.get("reason").asText());
    }
}
