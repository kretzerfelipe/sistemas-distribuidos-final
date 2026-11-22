package com.futhub.payment.charge;

import java.time.Instant;
import java.util.UUID;

public record Charge(
        UUID id,
        UUID subscriptionId,
        UUID userId,
        int amountCents,
        ChargeStatus status,
        int attempts,
        boolean refundPending,
        String failureReason,
        Instant createdAt,
        Instant updatedAt) {
}
