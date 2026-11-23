package com.futhub.payment.gateway;

import static org.assertj.core.api.Assertions.assertThatCode;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.util.UUID;

import com.futhub.payment.config.PaymentProperties;
import org.junit.jupiter.api.Test;

class SimulatedPaymentGatewayTest {

    private final GatewayRequest request = new GatewayRequest(UUID.randomUUID(), UUID.randomUUID(), 1990);

    @Test
    void alwaysDeclinesWhenTheFailureRateIsOne() {
        SimulatedPaymentGateway gateway = gatewayWith(0, 0, 1.0);

        assertThatThrownBy(() -> gateway.charge(request)).isInstanceOf(GatewayDeclinedException.class);
    }

    @Test
    void neverDeclinesWhenTheFailureRateIsZero() {
        SimulatedPaymentGateway gateway = gatewayWith(0, 0, 0.0);

        assertThatCode(() -> gateway.charge(request)).doesNotThrowAnyException();
    }

    @Test
    void invokerReportsTimeoutWhenTheGatewayIsSlowerThanTheLimit() {
        PaymentProperties properties = properties(2000, 0, 0.0, 50);
        GatewayInvoker invoker = new GatewayInvoker(new SimulatedPaymentGateway(properties), properties);

        org.assertj.core.api.Assertions.assertThat(invoker.invoke(request)).isEqualTo(GatewayOutcome.TIMEOUT);
        invoker.shutdown();
    }

    @Test
    void invokerReportsDeclinedWhenTheGatewayFails() {
        PaymentProperties properties = properties(0, 0, 1.0, 1000);
        GatewayInvoker invoker = new GatewayInvoker(new SimulatedPaymentGateway(properties), properties);

        org.assertj.core.api.Assertions.assertThat(invoker.invoke(request)).isEqualTo(GatewayOutcome.DECLINED);
        invoker.shutdown();
    }

    @Test
    void invokerReportsApprovedWhenTheGatewayAnswersInTime() {
        PaymentProperties properties = properties(0, 0, 0.0, 1000);
        GatewayInvoker invoker = new GatewayInvoker(new SimulatedPaymentGateway(properties), properties);

        org.assertj.core.api.Assertions.assertThat(invoker.invoke(request)).isEqualTo(GatewayOutcome.APPROVED);
        invoker.shutdown();
    }

    private SimulatedPaymentGateway gatewayWith(long latencyMs, long jitterMs, double failureRate) {
        return new SimulatedPaymentGateway(properties(latencyMs, jitterMs, failureRate, 1000));
    }

    private PaymentProperties properties(long latencyMs, long jitterMs, double failureRate, long timeoutMs) {
        return new PaymentProperties("", "", null,
                new PaymentProperties.Gateway(latencyMs, jitterMs, failureRate, timeoutMs, 3, 0), null);
    }
}
