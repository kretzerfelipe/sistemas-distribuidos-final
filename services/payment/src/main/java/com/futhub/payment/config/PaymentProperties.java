package com.futhub.payment.config;

import org.springframework.boot.context.properties.ConfigurationProperties;

@ConfigurationProperties(prefix = "payment")
public record PaymentProperties(String databaseUrl, String contractsDir, Broker broker, Gateway gateway, Outbox outbox) {

    public record Broker(
            String url,
            String exchange,
            String consumerQueue,
            String retryQueue,
            String deadLetterQueue,
            int maxRetries,
            int prefetch,
            int concurrency,
            long confirmTimeoutMs) {
    }

    public record Gateway(
            long latencyMs,
            long latencyJitterMs,
            double failureRate,
            long timeoutMs,
            int maxAttempts,
            long retryBackoffMs) {
    }

    public record Outbox(long pollIntervalMs, int batchSize) {
    }
}
