package com.futhub.payment.charge;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyInt;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.times;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import java.time.Instant;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.futhub.payment.config.PaymentProperties;
import com.futhub.payment.contract.Envelope;
import com.futhub.payment.gateway.GatewayInvoker;
import com.futhub.payment.gateway.GatewayOutcome;
import com.futhub.payment.gateway.GatewayRequest;
import com.futhub.payment.outbox.OutboxWriter;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.ArgumentCaptor;
import org.mockito.Captor;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import org.mockito.junit.jupiter.MockitoSettings;
import org.mockito.quality.Strictness;
import org.springframework.transaction.support.TransactionOperations;

@ExtendWith(MockitoExtension.class)
@MockitoSettings(strictness = Strictness.LENIENT)
class PaymentRequestProcessorTest {

    private static final int MAX_ATTEMPTS = 3;

    private final UUID messageId = UUID.randomUUID();
    private final UUID correlationId = UUID.randomUUID();
    private final UUID subscriptionId = UUID.randomUUID();
    private final UUID userId = UUID.randomUUID();
    private final UUID chargeId = UUID.randomUUID();

    @Mock
    private ChargeRepository charges;
    @Mock
    private ProcessedMessageRepository processedMessages;
    @Mock
    private OutboxWriter outbox;
    @Mock
    private ChargeLocks locks;
    @Mock
    private GatewayInvoker gateway;
    @Captor
    private ArgumentCaptor<Map<String, Object>> payload;

    private PaymentRequestProcessor processor;

    @BeforeEach
    void setUp() {
        PaymentProperties properties = new PaymentProperties(
                "", "", null, new PaymentProperties.Gateway(0, 0, 0, 1000, MAX_ATTEMPTS, 0), null);
        processor = new PaymentRequestProcessor(
                charges, processedMessages, outbox, locks, gateway, TransactionOperations.withoutTransaction(), properties);

        when(processedMessages.exists(messageId)).thenReturn(false);
        when(processedMessages.markProcessed(messageId)).thenReturn(true);
        ChargeLocks.Lock lock = () -> { };
        when(locks.tryAcquire(subscriptionId)).thenReturn(Optional.of(lock));
        when(charges.findBySubscriptionId(subscriptionId)).thenReturn(Optional.of(charge(ChargeStatus.PROCESSING, 0, false)));
        when(charges.findById(chargeId)).thenReturn(Optional.of(charge(ChargeStatus.PROCESSING, 0, false)));
        when(charges.reject(eq(chargeId), any())).thenReturn(true);
    }

    @Test
    void approvedChargePublishesPaymentApproved() {
        when(gateway.invoke(any(GatewayRequest.class))).thenReturn(GatewayOutcome.APPROVED);
        when(charges.approve(chargeId)).thenReturn(Optional.of(ChargeStatus.APPROVED));

        processor.process(envelope());

        verify(gateway, times(1)).invoke(any(GatewayRequest.class));
        verify(outbox).write(eq(correlationId), eq("PaymentApproved"), eq("payment.approved"), payload.capture());
        assertThat(payload.getValue())
                .containsEntry("subscriptionId", subscriptionId)
                .containsEntry("userId", userId)
                .containsEntry("chargeId", chargeId);
        verify(outbox, never()).write(any(), eq("RefundCompleted"), any(), any());
    }

    @Test
    void declinedChargeIsRejectedAfterTheLastAttempt() {
        when(gateway.invoke(any(GatewayRequest.class))).thenReturn(GatewayOutcome.DECLINED);

        processor.process(envelope());

        verify(gateway, times(MAX_ATTEMPTS)).invoke(any(GatewayRequest.class));
        verify(charges).recordAttempt(chargeId, 1);
        verify(charges).recordAttempt(chargeId, 2);
        verify(charges).recordAttempt(chargeId, 3);
        verify(charges).reject(chargeId, "GATEWAY_DECLINED");
        verify(outbox).write(eq(correlationId), eq("PaymentRejected"), eq("payment.rejected"), payload.capture());
        assertThat(payload.getValue()).containsEntry("reason", "GATEWAY_DECLINED").containsEntry("attempts", MAX_ATTEMPTS);
        verify(charges, never()).approve(any());
    }

    @Test
    void timeoutsAreRejectedWithTheTimeoutReason() {
        when(gateway.invoke(any(GatewayRequest.class))).thenReturn(GatewayOutcome.TIMEOUT);

        processor.process(envelope());

        verify(charges).reject(chargeId, "GATEWAY_TIMEOUT");
    }

    @Test
    void chargeIsApprovedWhenALaterAttemptSucceeds() {
        when(gateway.invoke(any(GatewayRequest.class)))
                .thenReturn(GatewayOutcome.DECLINED, GatewayOutcome.APPROVED);
        when(charges.approve(chargeId)).thenReturn(Optional.of(ChargeStatus.APPROVED));

        processor.process(envelope());

        verify(gateway, times(2)).invoke(any(GatewayRequest.class));
        verify(outbox).write(eq(correlationId), eq("PaymentApproved"), eq("payment.approved"), payload.capture());
        assertThat(payload.getValue()).containsEntry("attempts", 2);
        verify(charges, never()).reject(any(), any());
    }

    @Test
    void lateApprovalOfACanceledSubscriptionIsRefundedAtOnce() {
        when(gateway.invoke(any(GatewayRequest.class))).thenReturn(GatewayOutcome.APPROVED);
        when(charges.approve(chargeId)).thenReturn(Optional.of(ChargeStatus.REFUNDED));

        processor.process(envelope());

        verify(outbox).write(eq(correlationId), eq("PaymentApproved"), eq("payment.approved"), any());
        verify(outbox).write(eq(correlationId), eq("RefundCompleted"), eq("refund.completed"), payload.capture());
        assertThat(payload.getValue()).containsEntry("chargeId", chargeId).containsEntry("subscriptionId", subscriptionId);
    }

    @Test
    void canceledSubscriptionIsNotChargedWhenTheRetryStarts() {
        when(charges.findById(chargeId)).thenReturn(Optional.of(charge(ChargeStatus.PROCESSING, 1, true)));

        processor.process(envelope());

        verifyNoInteractions(gateway);
        verify(charges).reject(chargeId, "SUBSCRIPTION_CANCELED");
    }

    @Test
    void attemptsConsumedByCrashedConsumersAreNotRepeated() {
        when(charges.findBySubscriptionId(subscriptionId))
                .thenReturn(Optional.of(charge(ChargeStatus.PROCESSING, MAX_ATTEMPTS, false)));

        processor.process(envelope());

        verifyNoInteractions(gateway);
        verify(charges).reject(chargeId, "GATEWAY_ATTEMPTS_EXHAUSTED");
    }

    @Test
    void duplicateMessageDoesNothing() {
        when(processedMessages.exists(messageId)).thenReturn(true);

        processor.process(envelope());

        verifyNoInteractions(gateway, locks, outbox);
    }

    @Test
    void finishedChargeIsNeverChargedAgain() {
        when(charges.findBySubscriptionId(subscriptionId))
                .thenReturn(Optional.of(charge(ChargeStatus.APPROVED, 1, false)));

        processor.process(envelope());

        verifyNoInteractions(gateway, outbox);
        verify(processedMessages).markProcessed(messageId);
    }

    @Test
    void anotherConsumerHoldingTheChargeSendsTheMessageToRetry() {
        when(locks.tryAcquire(subscriptionId)).thenReturn(Optional.empty());

        assertThatThrownBy(() -> processor.process(envelope())).isInstanceOf(ChargeInProgressException.class);

        verifyNoInteractions(gateway, outbox);
        verify(charges, never()).recordAttempt(any(), anyInt());
    }

    private Charge charge(ChargeStatus status, int attempts, boolean refundPending) {
        return new Charge(chargeId, subscriptionId, userId, 1990, status, attempts, refundPending, null,
                Instant.now(), Instant.now());
    }

    private Envelope envelope() {
        ObjectNode body = new ObjectMapper().createObjectNode()
                .put("subscriptionId", subscriptionId.toString())
                .put("userId", userId.toString())
                .put("amountCents", 1990);
        return new Envelope(messageId, correlationId, "PaymentRequested", 1, Instant.now(), body);
    }
}
