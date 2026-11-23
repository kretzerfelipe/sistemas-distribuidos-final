package com.futhub.payment.messaging;

import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyBoolean;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.ArgumentMatchers.eq;
import static org.mockito.Mockito.doAnswer;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.never;
import static org.mockito.Mockito.verify;

import com.futhub.payment.config.PaymentProperties;
import com.rabbitmq.client.Channel;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.amqp.core.Message;
import org.springframework.amqp.core.MessageProperties;
import org.springframework.amqp.rabbit.connection.CorrelationData;
import org.springframework.amqp.rabbit.core.RabbitTemplate;

class MessageMoverTest {

    private static final String DEAD_LETTER_QUEUE = "payment.dlq";
    private static final long DELIVERY_TAG = 7L;

    private final RabbitTemplate template = mock(RabbitTemplate.class);
    private final Channel channel = mock(Channel.class);
    private MessageMover mover;

    @BeforeEach
    void setUp() {
        PaymentProperties.Broker broker = new PaymentProperties.Broker(
                "", "futhub.events", "payment.q", "payment.retry", DEAD_LETTER_QUEUE, 3, 1, 1, 1000);
        PaymentProperties properties = new PaymentProperties("", "", broker, null, null);
        mover = new MessageMover(template, properties);

        doAnswer(invocation -> {
            CorrelationData correlation = invocation.getArgument(3);
            correlation.getFuture().complete(new CorrelationData.Confirm(true, null));
            return null;
        }).when(template).send(eq(""), eq(DEAD_LETTER_QUEUE), any(Message.class), any(CorrelationData.class));
    }

    @Test
    void movesAMessageThatHasNoAmqpMessageIdInsteadOfLoopingForever() throws Exception {
        Message withoutMessageId = new Message("{\"broken\":true}".getBytes(), new MessageProperties());

        mover.move(withoutMessageId, channel, DELIVERY_TAG, DEAD_LETTER_QUEUE, "invalid", false);

        verify(channel).basicAck(DELIVERY_TAG, false);
        verify(channel, never()).basicNack(anyLong(), anyBoolean(), anyBoolean());
    }

    @Test
    void movesAMessageThatHasAnAmqpMessageId() throws Exception {
        MessageProperties properties = new MessageProperties();
        properties.setMessageId("0b9e5b52-6a4c-4a45-9c0e-0f2f0f4b1c11");

        mover.move(new Message("{}".getBytes(), properties), channel, DELIVERY_TAG, DEAD_LETTER_QUEUE, "invalid", false);

        verify(channel).basicAck(DELIVERY_TAG, false);
    }

    @Test
    void returnsTheMessageToTheQueueWhenTheBrokerDoesNotConfirm() throws Exception {
        doAnswer(invocation -> {
            CorrelationData correlation = invocation.getArgument(3);
            correlation.getFuture().complete(new CorrelationData.Confirm(false, "nack"));
            return null;
        }).when(template).send(eq(""), eq(DEAD_LETTER_QUEUE), any(Message.class), any(CorrelationData.class));

        mover.move(new Message("{}".getBytes(), new MessageProperties()), channel, DELIVERY_TAG, DEAD_LETTER_QUEUE, "invalid", false);

        verify(channel).basicNack(DELIVERY_TAG, false, true);
        verify(channel, never()).basicAck(anyLong(), anyBoolean());
    }
}
