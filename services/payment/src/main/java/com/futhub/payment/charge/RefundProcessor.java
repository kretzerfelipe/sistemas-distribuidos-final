package com.futhub.payment.charge;

import java.time.Instant;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Optional;

import com.futhub.payment.contract.Envelope;
import com.futhub.payment.contract.Events;
import com.futhub.payment.contract.RefundRequested;
import com.futhub.payment.outbox.OutboxWriter;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

@Service
public class RefundProcessor {

    private static final Logger log = LoggerFactory.getLogger(RefundProcessor.class);

    private final ChargeRepository charges;
    private final ProcessedMessageRepository processedMessages;
    private final OutboxWriter outbox;
    private final TransactionOperations transaction;

    public RefundProcessor(
            ChargeRepository charges,
            ProcessedMessageRepository processedMessages,
            OutboxWriter outbox,
            TransactionOperations transaction) {
        this.charges = charges;
        this.processedMessages = processedMessages;
        this.outbox = outbox;
        this.transaction = transaction;
    }

    public void process(Envelope envelope) {
        RefundRequested request = RefundRequested.from(envelope.payload());
        transaction.executeWithoutResult(status -> {
            if (!processedMessages.markProcessed(envelope.messageId())) {
                log.info("duplicate message ignored");
                return;
            }

            Optional<Charge> refunded = charges.refund(request.chargeId());
            if (refunded.isPresent()) {
                writeRefundCompleted(envelope, refunded.get());
                return;
            }

            if (charges.markRefundPendingById(request.chargeId()) == 1) {
                log.info("charge still processing, refund will happen when it finishes chargeId={}", request.chargeId());
                return;
            }
            log.info("nothing to refund, charge is unknown or already refunded chargeId={}", request.chargeId());
        });
    }

    private void writeRefundCompleted(Envelope envelope, Charge charge) {
        Map<String, Object> payload = new LinkedHashMap<>();
        payload.put("subscriptionId", charge.subscriptionId());
        payload.put("chargeId", charge.id());
        payload.put("userId", charge.userId());
        payload.put("amountCents", charge.amountCents());
        payload.put("refundedAt", Instant.now().toString());
        outbox.write(envelope.correlationId(), Events.REFUND_COMPLETED, Events.REFUND_COMPLETED_KEY, payload);
        log.info("charge refunded chargeId={}", charge.id());
    }
}
