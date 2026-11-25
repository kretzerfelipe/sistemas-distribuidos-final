package com.futhub.payment.gateway;

import java.util.concurrent.ThreadLocalRandom;

import com.futhub.payment.config.PaymentProperties;
import org.springframework.stereotype.Component;

@Component
public class SimulatedPaymentGateway implements PaymentGateway {

    private final PaymentProperties.Gateway settings;

    public SimulatedPaymentGateway(PaymentProperties properties) {
        this.settings = properties.gateway();
    }

    @Override
    public void charge(GatewayRequest request) throws GatewayDeclinedException, InterruptedException {
        Thread.sleep(nextDelayMs());
        boolean fails = ThreadLocalRandom.current().nextDouble() < settings.failureRate();
        if (fails) {
            throw new GatewayDeclinedException("simulated gateway declined the charge " + request.chargeId());
        }
    }

    private long nextDelayMs() {
        long jitter = settings.latencyJitterMs();
        long offset = jitter > 0 ? ThreadLocalRandom.current().nextLong(-jitter, jitter + 1) : 0;
        return Math.max(0, settings.latencyMs() + offset);
    }
}
