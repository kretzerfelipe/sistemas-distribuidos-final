package com.futhub.payment.outbox;

import java.util.UUID;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import com.futhub.payment.config.PaymentProperties;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.core.MessageDeliveryMode;
import org.springframework.amqp.core.MessageProperties;
import org.springframework.amqp.rabbit.connection.CorrelationData;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.stereotype.Component;

@Component
public class EventPublisher {

    private static final String JSON = "application/json";

    private final RabbitTemplate template;
    private final String exchange;
    private final long confirmTimeoutMs;

    public EventPublisher(RabbitTemplate template, PaymentProperties properties) {
        this.template = template;
        this.exchange = properties.broker().exchange();
        this.confirmTimeoutMs = properties.broker().confirmTimeoutMs();
    }

    public void publish(String routingKey, UUID messageId, byte[] body) {
        MessageProperties properties = new MessageProperties();
        properties.setContentType(JSON);
        properties.setDeliveryMode(MessageDeliveryMode.PERSISTENT);
        properties.setMessageId(messageId.toString());

        CorrelationData correlation = new CorrelationData(messageId.toString());
        template.send(exchange, routingKey, new Message(body, properties), correlation);
        awaitConfirmation(correlation);
    }

    private void awaitConfirmation(CorrelationData correlation) {
        try {
            CorrelationData.Confirm confirm = correlation.getFuture().get(confirmTimeoutMs, TimeUnit.MILLISECONDS);
            if (!confirm.isAck()) {
                throw new IllegalStateException("broker rejected the message: " + confirm.getReason());
            }
        } catch (TimeoutException e) {
            throw new IllegalStateException("broker did not confirm the message in time", e);
        } catch (ExecutionException e) {
            throw new IllegalStateException("broker confirmation failed", e);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new IllegalStateException("interrupted while waiting for the broker confirmation", e);
        }
    }
}
