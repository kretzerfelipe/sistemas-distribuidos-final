package com.futhub.payment.contract;

import java.util.UUID;

import com.fasterxml.jackson.databind.JsonNode;

final class Payloads {

    private Payloads() {
    }

    static UUID uuid(JsonNode payload, String field) {
        return UUID.fromString(payload.get(field).asText());
    }
}
