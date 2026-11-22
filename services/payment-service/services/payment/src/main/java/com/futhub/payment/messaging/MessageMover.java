package com.futhub.payment.messaging;

import java.io.IOException;
import java.util.concurrent.ExecutionException;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.TimeoutException;

import com.futhub.payment.config.PaymentProperties;
import com.rabbitmq.client.Channel;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.core.MessageDeliveryMode;
import org.springframework.amqp.core.MessageProperties;
import org.springframework.amqp.rabbit.connection.CorrelationData;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.stereotype.Component;

@Component
public class MessageMover {

    static final String RETRY_COUNT_HEADER = "x-retry-count";
    static final String FAILURE_REASON_HEADER = "x-failure-reason";

    private static final Logger log = LoggerFactory.getLogger(MessageMover.class);
    private static final String DEATH_HEADER = "x-death";
    private static final int MAX_REASON_LENGTH = 500;

    private final RabbitTemplate template;
    private final long confirmTimeoutMs;

    public MessageMover(RabbitTemplate template, PaymentProperties properties) {
        this.template = template;
        this.confirmTimeoutMs = properties.broker().confirmTimeoutMs();
    }

    public void move(Message original, Channel channel, long deliveryTag, String queue, String reason, boolean countsAsRetry)
            throws IOException {
        int retries = retryCountOf(original);
        MessageProperties properties = copyProperties(original);
        properties.setHeader(RETRY_COUNT_HEADER, (long) (countsAsRetry ? retries + 1 : retries));
        properties.setHeader(FAILURE_REASON_HEADER, truncate(reason));

        try {
            publishAndConfirm(queue, new Message(original.getBody(), properties));
        } catch (RuntimeException e) {
            log.error("could not move message, returning it to the queue target={}", queue, e);
            channel.basicNack(deliveryTag, false, true);
            return;
        }
        channel.basicAck(deliveryTag, false);
    }

    public static int retryCountOf(Message message) {
        Object value = message.getMessageProperties().getHeaders().get(RETRY_COUNT_HEADER);
        return value instanceof Number number ? number.intValue() : 0;
    }

    private MessageProperties copyProperties(Message original) {
        MessageProperties source = original.getMessageProperties();
        MessageProperties copy = new MessageProperties();
        copy.setContentType(source.getContentType());
        copy.setMessageId(source.getMessageId());
        copy.setDeliveryMode(MessageDeliveryMode.PERSISTENT);
        source.getHeaders().forEach((key, value) -> {
            if (!DEATH_HEADER.equals(key)) {
                copy.setHeader(key, value);
            }
        });
        return copy;
    }

    private void publishAndConfirm(String queue, Message message) {
        CorrelationData correlation = new CorrelationData(message.getMessageProperties().getMessageId());
        template.send("", queue, message, correlation);
        try {
            CorrelationData.Confirm confirm = correlation.getFuture().get(confirmTimeoutMs, TimeUnit.MILLISECONDS);
            if (!confirm.isAck()) {
                throw new IllegalStateException("broker did not confirm the message: " + confirm.getReason());
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

    private static String truncate(String reason) {
        if (reason == null) {
            return "";
        }
        return reason.length() <= MAX_REASON_LENGTH ? reason : reason.substring(0, MAX_REASON_LENGTH);
    }
}
