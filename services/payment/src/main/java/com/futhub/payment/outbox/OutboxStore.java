package com.futhub.payment.outbox;

import java.time.OffsetDateTime;
import java.util.List;
import java.util.Map;
import java.util.UUID;

import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Repository;

@Repository
public class OutboxStore {

    private static final RowMapper<OutboxRow> MAPPER = (rs, rowNumber) -> new OutboxRow(
            rs.getObject("id", UUID.class),
            rs.getObject("correlation_id", UUID.class),
            rs.getString("type"),
            rs.getString("routing_key"),
            rs.getString("payload"),
            rs.getObject("created_at", OffsetDateTime.class).toInstant());

    private final NamedParameterJdbcTemplate jdbc;

    public OutboxStore(NamedParameterJdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public List<OutboxRow> lockUnpublished(int limit) {
        return jdbc.query("""
                SELECT id, correlation_id, type, routing_key, CAST(payload AS text) AS payload, created_at
                  FROM outbox
                 WHERE published_at IS NULL
                 ORDER BY position
                 LIMIT :limit
                   FOR UPDATE SKIP LOCKED""",
                Map.of("limit", limit), MAPPER);
    }

    public void markPublished(UUID id) {
        jdbc.update("UPDATE outbox SET published_at = now() WHERE id = :id", Map.of("id", id));
    }
}
