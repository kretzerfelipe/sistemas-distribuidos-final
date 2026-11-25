package com.futhub.payment.gateway;

import java.util.concurrent.ExecutionException;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.concurrent.Future;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import com.futhub.payment.config.PaymentProperties;
import jakarta.annotation.PreDestroy;
import org.springframework.stereotype.Component;

@Component
public class GatewayInvoker {

    private final PaymentGateway gateway;
    private final long timeoutMs;
    private final ExecutorService executor = Executors.newCachedThreadPool(runnable -> {
        Thread thread = new Thread(runnable, "gateway-call");
        thread.setDaemon(true);
        return thread;
    });

    public GatewayInvoker(PaymentGateway gateway, PaymentProperties properties) {
        this.gateway = gateway;
        this.timeoutMs = properties.gateway().timeoutMs();
    }

    public GatewayOutcome invoke(GatewayRequest request) {
        Future<Void> call = executor.submit(() -> {
            gateway.charge(request);
            return null;
        });
        try {
            call.get(timeoutMs, TimeUnit.MILLISECONDS);
            return GatewayOutcome.APPROVED;
        } catch (TimeoutException e) {
            call.cancel(true);
            return GatewayOutcome.TIMEOUT;
        } catch (ExecutionException e) {
            return e.getCause() instanceof GatewayDeclinedException ? GatewayOutcome.DECLINED : GatewayOutcome.ERROR;
        } catch (InterruptedException e) {
            call.cancel(true);
            Thread.currentThread().interrupt();
            throw new IllegalStateException("interrupted while waiting for the gateway", e);
        }
    }

    @PreDestroy
    void shutdown() {
        executor.shutdownNow();
    }
}
