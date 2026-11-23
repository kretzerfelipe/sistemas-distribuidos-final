package com.futhub.payment.gateway;

import java.util.UUID;

public record GatewayRequest(UUID chargeId, UUID subscriptionId, int amountCents) {
}
