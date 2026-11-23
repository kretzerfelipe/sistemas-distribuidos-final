package com.futhub.payment.charge;

import java.util.UUID;

public class ChargeInProgressException extends RuntimeException {

    public ChargeInProgressException(UUID subscriptionId) {
        super("charge of subscription " + subscriptionId + " is being processed by another consumer");
    }
}
