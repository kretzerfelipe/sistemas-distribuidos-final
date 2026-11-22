package com.futhub.payment.gateway;

public class GatewayDeclinedException extends Exception {

    public GatewayDeclinedException(String message) {
        super(message);
    }
}
