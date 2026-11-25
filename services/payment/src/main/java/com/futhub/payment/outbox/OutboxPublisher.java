package com.futhub.payment.outbox;

import java.time.Duration;
import java.time.Instant;
import java.util.List;
import java.util.concurrent.atomic.AtomicReference;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.futhub.payment.config.PaymentProperties;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.slf4j.MDC;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;
import org.springframework.transaction.support.TransactionOperations;

@Component
public class OutboxPublisher {

    private static final Logger log = LoggerFactory.getLogger(OutboxPublisher.class);
    private static final int ENVELOPE_VERSION = 1;
    private static final Duration INITIAL_BACKOFF = Duration.ofSeconds(1);
    private static final Duration MAX_BACKOFF = Duration.ofSeconds(30);

    private final OutboxStore store;
    private final EventPublisher publisher;
    private final TransactionOperations transaction;
    private final ObjectMapper mapper;
    private final int batchSize;

    private Duration backoff = INITIAL_BACKOFF;
    private Instant nextAttemptAt = Instant.EPOCH;
    private boolean failing;

    public OutboxPublisher(
            OutboxStore store,
            EventPublisher publisher,
            TransactionOperations transaction,
            ObjectMapper mapper,
            PaymentProperties properties) {
        this.store = store;
        this.publisher = publisher;
        this.transaction = transaction;
        this.mapper = mapper;
        this.batchSize = properties.outbox().batchSize();
    }

    @Scheduled(fixedDelayString = "${payment.outbox.poll-interval-ms}")
    public void drain() {
        if (Instant.now().isBefore(nextAttemptAt)) {
            return;
        }
        try {
            int published;
            do {
                published = publishBatch();
            } while (published == batchSize);
            reportRecovery();
        } catch (RuntimeException e) {
            reportFailure(e);
        }
    }

    private int publishBatch() {
        AtomicReference<RuntimeException> failure = new AtomicReference<>();
        Integer published = transaction.execute(status -> {
            List<OutboxRow> rows = store.lockUnpublished(batchSize);
            int count = 0;
            for (OutboxRow row : rows) {
                try {
                    publish(row);
                    store.markPublished(row.id());
                    count++;
                } catch (RuntimeException e) {
                    failure.set(e);
                    break;
                }
            }
            return count;
        });
        if (failure.get() != null) {
            throw failure.get();
        }
        return published == null ? 0 : published;
    }

    private void publish(OutboxRow row) {
        MDC.put("messageId", row.id().toString());
        MDC.put("correlationId", row.correlationId().toString());
        MDC.put("type", row.type());
        try {
            publisher.publish(row.routingKey(), row.id(), toEnvelope(row));
            log.info("event published routingKey={}", row.routingKey());
        } finally {
            MDC.remove("messageId");
            MDC.remove("correlationId");
            MDC.remove("type");
        }
    }

    private byte[] toEnvelope(OutboxRow row) {
        try {
            ObjectNode envelope = mapper.createObjectNode();
            envelope.put("messageId", row.id().toString());
            envelope.put("correlationId", row.correlationId().toString());
            envelope.put("type", row.type());
            envelope.put("version", ENVELOPE_VERSION);
            envelope.put("occurredAt", row.createdAt().toString());
            JsonNode payload = mapper.readTree(row.payload());
            envelope.set("payload", payload);
            return mapper.writeValueAsBytes(envelope);
        } catch (JsonProcessingException e) {
            throw new IllegalStateException("outbox row " + row.id() + " has an invalid payload", e);
        }
    }

    private void reportFailure(RuntimeException error) {
        if (!failing) {
            failing = true;
            log.warn("outbox could not publish, messages stay stored and will be retried retryIn={}", backoff, error);
        }
        nextAttemptAt = Instant.now().plus(backoff);
        backoff = backoff.multipliedBy(2).compareTo(MAX_BACKOFF) > 0 ? MAX_BACKOFF : backoff.multipliedBy(2);
    }

    private void reportRecovery() {
        backoff = INITIAL_BACKOFF;
        nextAttemptAt = Instant.EPOCH;
        if (failing) {
            failing = false;
            log.info("outbox publishing recovered");
        }
    }
}
