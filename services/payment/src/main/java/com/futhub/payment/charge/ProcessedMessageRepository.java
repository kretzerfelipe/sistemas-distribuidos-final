package com.futhub.payment.charge;

import java.util.Map;
import java.util.UUID;

import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Repository;

@Repository
public class ProcessedMessageRepository {

    private final NamedParameterJdbcTemplate jdbc;

    public ProcessedMessageRepository(NamedParameterJdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public boolean exists(UUID messageId) {
        Boolean found = jdbc.queryForObject(
                "SELECT EXISTS (SELECT 1 FROM processed_messages WHERE message_id = :messageId)",
                Map.of("messageId", messageId),
                Boolean.class);
        return Boolean.TRUE.equals(found);
    }

    public boolean markProcessed(UUID messageId) {
        int inserted = jdbc.update(
                "INSERT INTO processed_messages (message_id) VALUES (:messageId) ON CONFLICT DO NOTHING",
                Map.of("messageId", messageId));
        return inserted == 1;
    }
}
