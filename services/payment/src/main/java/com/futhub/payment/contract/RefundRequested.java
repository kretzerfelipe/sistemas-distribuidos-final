package com.futhub.payment.contract;

import java.util.UUID;

import com.fasterxml.jackson.databind.JsonNode;

public record RefundRequested(UUID subscriptionId, UUID chargeId) {

    public static RefundRequested from(JsonNode payload) {
        return new RefundRequested(Payloads.uuid(payload, "subscriptionId"), Payloads.uuid(payload, "chargeId"));
    }
}
