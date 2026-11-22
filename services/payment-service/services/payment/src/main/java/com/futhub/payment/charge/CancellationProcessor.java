package com.futhub.payment.charge;

import com.futhub.payment.contract.Envelope;
import com.futhub.payment.contract.SubscriptionCanceled;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Service;
import org.springframework.transaction.support.TransactionOperations;

@Service
public class CancellationProcessor {

    private static final Logger log = LoggerFactory.getLogger(CancellationProcessor.class);

    private final ChargeRepository charges;
    private final ProcessedMessageRepository processedMessages;
    private final TransactionOperations transaction;

    public CancellationProcessor(
            ChargeRepository charges,
            ProcessedMessageRepository processedMessages,
            TransactionOperations transaction) {
        this.charges = charges;
        this.processedMessages = processedMessages;
        this.transaction = transaction;
    }

    public void process(Envelope envelope) {
        SubscriptionCanceled canceled = SubscriptionCanceled.from(envelope.payload());
        transaction.executeWithoutResult(status -> {
            if (!processedMessages.markProcessed(envelope.messageId())) {
                log.info("duplicate message ignored");
                return;
            }

            int marked = charges.markRefundPendingBySubscription(canceled.subscriptionId());
            if (marked == 1) {
                log.info("charge still processing, it will be refunded when it finishes subscriptionId={} reason={}",
                        canceled.subscriptionId(), canceled.reason());
                return;
            }
            log.info("no charge in progress to compensate subscriptionId={} reason={}",
                    canceled.subscriptionId(), canceled.reason());
        });
    }
}
