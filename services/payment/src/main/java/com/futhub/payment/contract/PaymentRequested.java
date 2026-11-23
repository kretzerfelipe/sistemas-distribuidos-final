package com.futhub.payment.contract;

import java.util.UUID;

import com.fasterxml.jackson.databind.JsonNode;

public record PaymentRequested(UUID subscriptionId, UUID userId, int amountCents) {

    public static PaymentRequested from(JsonNode payload) {
        return new PaymentRequested(
                Payloads.uuid(payload, "subscriptionId"),
                Payloads.uuid(payload, "userId"),
                payload.get("amountCents").asInt());
    }
}
