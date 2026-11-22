package com.futhub.payment.outbox;

import java.util.Map;
import java.util.UUID;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;
import org.springframework.jdbc.core.namedparam.MapSqlParameterSource;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Component;

@Component
public class OutboxWriter {

    private final NamedParameterJdbcTemplate jdbc;
    private final ObjectMapper mapper;

    public OutboxWriter(NamedParameterJdbcTemplate jdbc, ObjectMapper mapper) {
        this.jdbc = jdbc;
        this.mapper = mapper;
    }

    public UUID write(UUID correlationId, String type, String routingKey, Map<String, Object> payload) {
        UUID messageId = UUID.randomUUID();
        jdbc.update("""
                INSERT INTO outbox (id, correlation_id, type, routing_key, payload)
                VALUES (:id, :correlationId, :type, :routingKey, CAST(:payload AS jsonb))""",
                new MapSqlParameterSource()
                        .addValue("id", messageId)
                        .addValue("correlationId", correlationId)
                        .addValue("type", type)
                        .addValue("routingKey", routingKey)
                        .addValue("payload", serialize(payload)));
        return messageId;
    }

    private String serialize(Map<String, Object> payload) {
        try {
            return mapper.writeValueAsString(payload);
        } catch (JsonProcessingException e) {
            throw new IllegalArgumentException("payload cannot be serialized", e);
        }
    }
}
