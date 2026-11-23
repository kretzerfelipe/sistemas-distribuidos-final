package com.futhub.payment.charge;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;
import static org.mockito.Mockito.verifyNoInteractions;
import static org.mockito.Mockito.when;

import java.time.Instant;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;
import com.futhub.payment.contract.Envelope;
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
class RefundProcessorTest {

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
    @Captor
    private ArgumentCaptor<Map<String, Object>> payload;

    private RefundProcessor processor;

    @BeforeEach
    void setUp() {
        processor = new RefundProcessor(charges, processedMessages, outbox, TransactionOperations.withoutTransaction());
        when(processedMessages.markProcessed(messageId)).thenReturn(true);
    }

    @Test
    void approvedChargeIsRefundedAndPublishesRefundCompleted() {
        Charge refunded = new Charge(chargeId, subscriptionId, userId, 1990, ChargeStatus.REFUNDED, 1, false, null,
                Instant.now(), Instant.now());
        when(charges.refund(chargeId)).thenReturn(Optional.of(refunded));

        processor.process(envelope());

        verify(outbox).write(eq(correlationId), eq("RefundCompleted"), eq("refund.completed"), payload.capture());
        assertThat(payload.getValue()).containsEntry("chargeId", chargeId).containsEntry("subscriptionId", subscriptionId);
    }

    @Test
    void chargeStillProcessingIsMarkedToBeRefundedWhenItFinishes() {
        when(charges.refund(chargeId)).thenReturn(Optional.empty());
        when(charges.markRefundPendingById(chargeId)).thenReturn(1);

        processor.process(envelope());

        verify(charges).markRefundPendingById(chargeId);
        verify(outbox, never()).write(any(), any(), any(), any());
    }

    @Test
    void alreadyRefundedChargePublishesNothing() {
        when(charges.refund(chargeId)).thenReturn(Optional.empty());
        when(charges.markRefundPendingById(chargeId)).thenReturn(0);

        processor.process(envelope());

        verify(outbox, never()).write(any(), any(), any(), any());
    }

    @Test
    void duplicateMessageDoesNothing() {
        when(processedMessages.markProcessed(messageId)).thenReturn(false);

        processor.process(envelope());

        verifyNoInteractions(charges, outbox);
    }

    private Envelope envelope() {
        ObjectNode body = new ObjectMapper().createObjectNode()
                .put("subscriptionId", subscriptionId.toString())
                .put("chargeId", chargeId.toString());
        return new Envelope(messageId, correlationId, "RefundRequested", 1, Instant.now(), body);
    }
}
