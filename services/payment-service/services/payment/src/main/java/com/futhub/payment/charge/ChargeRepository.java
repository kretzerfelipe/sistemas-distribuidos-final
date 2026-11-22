package com.futhub.payment.charge;

import java.time.OffsetDateTime;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import org.springframework.jdbc.core.RowMapper;
import org.springframework.jdbc.core.namedparam.MapSqlParameterSource;
import org.springframework.jdbc.core.namedparam.NamedParameterJdbcTemplate;
import org.springframework.stereotype.Repository;

@Repository
public class ChargeRepository {

    private static final String COLUMNS = """
            id, subscription_id, user_id, amount_cents, status, attempts, refund_pending,
            failure_reason, created_at, updated_at""";

    private static final RowMapper<Charge> MAPPER = (rs, rowNumber) -> new Charge(
            rs.getObject("id", UUID.class),
            rs.getObject("subscription_id", UUID.class),
            rs.getObject("user_id", UUID.class),
            rs.getInt("amount_cents"),
            ChargeStatus.valueOf(rs.getString("status")),
            rs.getInt("attempts"),
            rs.getBoolean("refund_pending"),
            rs.getString("failure_reason"),
            rs.getObject("created_at", OffsetDateTime.class).toInstant(),
            rs.getObject("updated_at", OffsetDateTime.class).toInstant());

    private final NamedParameterJdbcTemplate jdbc;

    public ChargeRepository(NamedParameterJdbcTemplate jdbc) {
        this.jdbc = jdbc;
    }

    public boolean insertProcessing(UUID id, UUID subscriptionId, UUID userId, int amountCents) {
        int inserted = jdbc.update("""
                INSERT INTO charges (id, subscription_id, user_id, amount_cents, status)
                VALUES (:id, :subscriptionId, :userId, :amountCents, 'PROCESSING')
                ON CONFLICT (subscription_id) DO NOTHING""",
                new MapSqlParameterSource()
                        .addValue("id", id)
                        .addValue("subscriptionId", subscriptionId)
                        .addValue("userId", userId)
                        .addValue("amountCents", amountCents));
        return inserted == 1;
    }

    public Optional<Charge> findById(UUID id) {
        return jdbc.query("SELECT " + COLUMNS + " FROM charges WHERE id = :id", Map.of("id", id), MAPPER)
                .stream().findFirst();
    }

    public Optional<Charge> findBySubscriptionId(UUID subscriptionId) {
        return jdbc.query(
                        "SELECT " + COLUMNS + " FROM charges WHERE subscription_id = :subscriptionId",
                        Map.of("subscriptionId", subscriptionId), MAPPER)
                .stream().findFirst();
    }

    public void recordAttempt(UUID id, int attempts) {
        jdbc.update(
                "UPDATE charges SET attempts = :attempts, updated_at = now() WHERE id = :id AND status = 'PROCESSING'",
                Map.of("id", id, "attempts", attempts));
    }

    public Optional<ChargeStatus> approve(UUID id) {
        return jdbc.query("""
                        UPDATE charges
                           SET status = CASE WHEN refund_pending THEN 'REFUNDED' ELSE 'APPROVED' END,
                               updated_at = now()
                         WHERE id = :id AND status = 'PROCESSING'
                     RETURNING status""",
                Map.of("id", id),
                (rs, rowNumber) -> ChargeStatus.valueOf(rs.getString("status")))
                .stream().findFirst();
    }

    public boolean reject(UUID id, String reason) {
        int updated = jdbc.update("""
                UPDATE charges
                   SET status = 'REJECTED', failure_reason = :reason, updated_at = now()
                 WHERE id = :id AND status = 'PROCESSING'""",
                Map.of("id", id, "reason", reason));
        return updated == 1;
    }

    public Optional<Charge> refund(UUID id) {
        String sql = "UPDATE charges SET status = 'REFUNDED', updated_at = now() "
                + "WHERE id = :id AND status = 'APPROVED' RETURNING " + COLUMNS;
        return jdbc.query(sql, Map.of("id", id), MAPPER).stream().findFirst();
    }

    public int markRefundPendingById(UUID id) {
        return jdbc.update("""
                UPDATE charges SET refund_pending = true, updated_at = now()
                 WHERE id = :id AND status = 'PROCESSING'""",
                Map.of("id", id));
    }

    public int markRefundPendingBySubscription(UUID subscriptionId) {
        return jdbc.update("""
                UPDATE charges SET refund_pending = true, updated_at = now()
                 WHERE subscription_id = :subscriptionId AND status = 'PROCESSING'""",
                Map.of("subscriptionId", subscriptionId));
    }
}
