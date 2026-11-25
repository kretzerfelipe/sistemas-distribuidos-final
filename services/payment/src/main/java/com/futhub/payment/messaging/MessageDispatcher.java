package com.futhub.payment.messaging;

import com.futhub.payment.charge.CancellationProcessor;
import com.futhub.payment.charge.PaymentRequestProcessor;
import com.futhub.payment.charge.RefundProcessor;
import com.futhub.payment.contract.ContractValidator;
import com.futhub.payment.contract.Envelope;
import com.futhub.payment.contract.Events;
import com.futhub.payment.contract.InvalidMessageException;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class MessageDispatcher {

    private static final Logger log = LoggerFactory.getLogger(MessageDispatcher.class);

    private final ContractValidator validator;
    private final PaymentRequestProcessor paymentRequests;
    private final RefundProcessor refunds;
    private final CancellationProcessor cancellations;

    public MessageDispatcher(
            ContractValidator validator,
            PaymentRequestProcessor paymentRequests,
            RefundProcessor refunds,
            CancellationProcessor cancellations) {
        this.validator = validator;
        this.paymentRequests = paymentRequests;
        this.refunds = refunds;
        this.cancellations = cancellations;
    }

    public void handle(byte[] body) {
        Envelope envelope = validator.validate(body);
        log.info("message received");
        switch (envelope.type()) {
            case Events.PAYMENT_REQUESTED -> paymentRequests.process(envelope);
            case Events.REFUND_REQUESTED -> refunds.process(envelope);
            case Events.SUBSCRIPTION_CANCELED -> cancellations.process(envelope);
            default -> throw new InvalidMessageException("payment does not handle messages of type " + envelope.type());
        }
    }
}
