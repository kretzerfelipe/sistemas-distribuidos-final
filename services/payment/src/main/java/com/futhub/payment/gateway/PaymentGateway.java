package com.futhub.payment.gateway;

public interface PaymentGateway {

    void charge(GatewayRequest request) throws GatewayDeclinedException, InterruptedException;
}
