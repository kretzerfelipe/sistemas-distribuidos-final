package com.futhub.payment.contract;

import java.io.IOException;
import java.util.List;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.slf4j.MDC;

public final class MessageContext implements AutoCloseable {

    private static final List<String> FIELDS = List.of("messageId", "correlationId", "type");

    private MessageContext() {
    }

    public static MessageContext open(byte[] body, ObjectMapper mapper) {
        try {
            JsonNode tree = mapper.readTree(body);
            if (tree != null && tree.isObject()) {
                for (String field : FIELDS) {
                    JsonNode value = tree.get(field);
                    if (value != null && value.isTextual()) {
                        MDC.put(field, value.asText());
                    }
                }
            }
        } catch (IOException ignored) {
            return new MessageContext();
        }
        return new MessageContext();
    }

    @Override
    public void close() {
        FIELDS.forEach(MDC::remove);
    }
}
