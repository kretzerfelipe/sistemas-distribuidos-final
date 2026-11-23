package com.futhub.payment.charge;

import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.UUID;

import com.futhub.payment.config.PaymentProperties;
import com.futhub.payment.contract.Envelope;
import com.futhub.payment.contract.Events;
import com.futhub.payment.contract.PaymentRequested;
import com.futhub.payment.gateway.GatewayInvoker;
import com.futhub.payment.gateway.GatewayOutcome;
import com.futhub.payment.gateway.GatewayRequest;
import com.futhub.payment.outbox.OutboxWriter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

@Service
public class PaymentRequestProcessor {

    static final String SUBSCRIPTION_CANCELED_REASON = "SUBSCRIPTION_CANCELED";
    static final String ATTEMPTS_EXHAUSTED_REASON = "GATEWAY_ATTEMPTS_EXHAUSTED";

    private static final Logger log = LoggerFactory.getLogger(PaymentRequestProcessor.class);

    private final ChargeRepository charges;
    private final ProcessedMessageRepository processedMessages;
    private final OutboxWriter outbox;
    private final ChargeLocks locks;
    private final GatewayInvoker gateway;
    private final TransactionOperations transaction;
    private final PaymentProperties.Gateway settings;

    public PaymentRequestProcessor(
            ChargeRepository charges,
            ProcessedMessageRepository processedMessages,
            OutboxWriter outbox,
            ChargeLocks locks,
            GatewayInvoker gateway,
            TransactionOperations transaction,
            PaymentProperties properties) {
        this.charges = charges;
        this.processedMessages = processedMessages;
        this.outbox = outbox;
        this.locks = locks;
        this.gateway = gateway;
        this.transaction = transaction;
        this.settings = properties.gateway();
    }

    public void process(Envelope envelope) {
        if (processedMessages.exists(envelope.messageId())) {
            log.info("duplicate message ignored");
            return;
        }

        PaymentRequested request = PaymentRequested.from(envelope.payload());
        ChargeLocks.Lock lock = locks.tryAcquire(request.subscriptionId())
                .orElseThrow(() -> new ChargeInProgressException(request.subscriptionId()));
        try (lock) {
            Charge charge = register(request);
            if (charge.status() != ChargeStatus.PROCESSING) {
                markProcessed(envelope);
                log.info("charge already finished status={} chargeId={}", charge.status(), charge.id());
                return;
            }
            Decision decision = decide(charge);
            finish(envelope, charge, decision);
        }
    }

    private Charge register(PaymentRequested request) {
        return transaction.execute(status -> {
            charges.insertProcessing(UUID.randomUUID(), request.subscriptionId(), request.userId(), request.amountCents());
            return charges.findBySubscriptionId(request.subscriptionId()).orElseThrow();
        });
    }

    private Decision decide(Charge charge) {
        int attempts = charge.attempts();
        GatewayOutcome lastOutcome = null;

        while (attempts < settings.maxAttempts()) {
            if (isSubscriptionCanceled(charge)) {
                return Decision.rejected(attempts, SUBSCRIPTION_CANCELED_REASON);
            }
            attempts++;
            charges.recordAttempt(charge.id(), attempts);
            lastOutcome = gateway.invoke(new GatewayRequest(charge.id(), charge.subscriptionId(), charge.amountCents()));
            log.info("gateway answered outcome={} attempt={} chargeId={}", lastOutcome, attempts, charge.id());
            if (lastOutcome.isApproved()) {
                return Decision.approved(attempts);
            }
            if (attempts < settings.maxAttempts()) {
                pauseBeforeNextAttempt();
            }
        }

        String reason = lastOutcome == null ? ATTEMPTS_EXHAUSTED_REASON : lastOutcome.reason();
        return Decision.rejected(attempts, reason);
    }

    private boolean isSubscriptionCanceled(Charge charge) {
        return charges.findById(charge.id()).map(Charge::refundPending).orElse(false);
    }

    private void pauseBeforeNextAttempt() {
        if (settings.retryBackoffMs() <= 0) {
            return;
        }
        try {
            Thread.sleep(settings.retryBackoffMs());
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new IllegalStateException("interrupted between gateway attempts", e);
        }
    }

    private void markProcessed(Envelope envelope) {
        transaction.executeWithoutResult(status -> processedMessages.markProcessed(envelope.messageId()));
    }

    private void finish(Envelope envelope, Charge charge, Decision decision) {
        transaction.executeWithoutResult(status -> {
            if (!processedMessages.markProcessed(envelope.messageId())) {
                log.info("duplicate message ignored");
                return;
            }
            if (decision.approved()) {
                approve(envelope, charge, decision);
            } else {
                reject(envelope, charge, decision);
            }
        });
    }

    private void approve(Envelope envelope, Charge charge, Decision decision) {
        ChargeStatus finalStatus = charges.approve(charge.id()).orElse(null);
        if (finalStatus == null) {
            log.warn("charge was already finalized, approval dropped chargeId={}", charge.id());
            return;
        }

        Map<String, Object> approved = new LinkedHashMap<>();
        approved.put("subscriptionId", charge.subscriptionId());
        approved.put("userId", charge.userId());
        approved.put("chargeId", charge.id());
        approved.put("amountCents", charge.amountCents());
        approved.put("attempts", decision.attempts());
        outbox.write(envelope.correlationId(), Events.PAYMENT_APPROVED, Events.PAYMENT_APPROVED_KEY, approved);
        log.info("charge approved chargeId={} finalStatus={}", charge.id(), finalStatus);

        if (finalStatus == ChargeStatus.REFUNDED) {
            writeRefundCompleted(envelope.correlationId(), charge);
        }
    }

    private void reject(Envelope envelope, Charge charge, Decision decision) {
        if (!charges.reject(charge.id(), decision.reason())) {
            log.warn("charge was already finalized, rejection dropped chargeId={}", charge.id());
            return;
        }

        Map<String, Object> rejected = new LinkedHashMap<>();
        rejected.put("subscriptionId", charge.subscriptionId());
        rejected.put("userId", charge.userId());
        rejected.put("reason", decision.reason());
        rejected.put("attempts", decision.attempts());
        outbox.write(envelope.correlationId(), Events.PAYMENT_REJECTED, Events.PAYMENT_REJECTED_KEY, rejected);
        log.info("charge rejected chargeId={} reason={}", charge.id(), decision.reason());
    }

    private void writeRefundCompleted(UUID correlationId, Charge charge) {
        Map<String, Object> refunded = new LinkedHashMap<>();
        refunded.put("subscriptionId", charge.subscriptionId());
        refunded.put("chargeId", charge.id());
        refunded.put("userId", charge.userId());
        refunded.put("amountCents", charge.amountCents());
        refunded.put("refundedAt", Instant.now().toString());
        outbox.write(correlationId, Events.REFUND_COMPLETED, Events.REFUND_COMPLETED_KEY, refunded);
        log.info("approved charge refunded right away because the subscription was canceled chargeId={}", charge.id());
    }

    private record Decision(boolean approved, int attempts, String reason) {

        static Decision approved(int attempts) {
            return new Decision(true, attempts, null);
        }

        static Decision rejected(int attempts, String reason) {
            return new Decision(false, attempts, reason);
        }
    }
}
