package com.futhub.payment.messaging;

import java.io.IOException;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.futhub.payment.charge.ChargeInProgressException;
import com.futhub.payment.config.PaymentProperties;
import com.futhub.payment.config.RabbitConfig;
import com.futhub.payment.contract.InvalidMessageException;
import com.futhub.payment.contract.MessageContext;
import com.rabbitmq.client.Channel;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.rabbit.annotation.RabbitListener;
import org.springframework.amqp.support.AmqpHeaders;
import org.springframework.messaging.handler.annotation.Header;
import org.springframework.stereotype.Component;

@Component
public class PaymentConsumer {

    private static final Logger log = LoggerFactory.getLogger(PaymentConsumer.class);

    private final MessageDispatcher dispatcher;
    private final MessageMover mover;
    private final ObjectMapper mapper;
    private final PaymentProperties.Broker broker;

    public PaymentConsumer(
            MessageDispatcher dispatcher, MessageMover mover, ObjectMapper mapper, PaymentProperties properties) {
        this.dispatcher = dispatcher;
        this.mover = mover;
        this.mapper = mapper;
        this.broker = properties.broker();
    }

    @RabbitListener(queues = "${payment.broker.consumer-queue}", containerFactory = RabbitConfig.LISTENER_FACTORY)
    public void onMessage(Message message, Channel channel, @Header(AmqpHeaders.DELIVERY_TAG) long deliveryTag)
            throws IOException {
        try (MessageContext ignored = MessageContext.open(message.getBody(), mapper)) {
            handle(message, channel, deliveryTag);
        }
    }

    private void handle(Message message, Channel channel, long deliveryTag) throws IOException {
        try {
            dispatcher.handle(message.getBody());
            channel.basicAck(deliveryTag, false);
        } catch (InvalidMessageException e) {
            log.error("invalid message, sending to the dead letter queue: {}", e.getMessage());
            mover.move(message, channel, deliveryTag, broker.deadLetterQueue(), e.getMessage(), false);
        } catch (ChargeInProgressException e) {
            log.info("{}, it will be checked again shortly", e.getMessage());
            mover.move(message, channel, deliveryTag, broker.retryQueue(), e.getMessage(), false);
        } catch (RuntimeException e) {
            int retries = MessageMover.retryCountOf(message);
            if (retries >= broker.maxRetries()) {
                log.error("message failed too many times, sending to the dead letter queue retries={}", retries, e);
                mover.move(message, channel, deliveryTag, broker.deadLetterQueue(), describe(e), true);
                return;
            }
            log.warn("message could not be processed, it will be retried retries={}", retries, e);
            mover.move(message, channel, deliveryTag, broker.retryQueue(), describe(e), true);
        }
    }

    private static String describe(RuntimeException error) {
        return error.getClass().getSimpleName() + ": " + error.getMessage();
    }
}
