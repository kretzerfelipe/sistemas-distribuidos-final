package com.futhub.payment.config;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.amqp.core.AcknowledgeMode;
import org.springframework.amqp.rabbit.config.SimpleRabbitListenerContainerFactory;
import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;
import org.springframework.amqp.rabbit.core.RabbitTemplate;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class RabbitConfig {

    public static final String LISTENER_FACTORY = "paymentListenerFactory";

    private static final Logger log = LoggerFactory.getLogger(RabbitConfig.class);
    private static final int CONNECTION_TIMEOUT_MS = 3000;
    private static final int HEARTBEAT_SECONDS = 10;
    private static final String DEFAULT_VIRTUAL_HOST = "/";

    @Bean
    public CachingConnectionFactory connectionFactory(PaymentProperties properties) {
        CachingConnectionFactory factory = new CachingConnectionFactory();
        factory.setUri(properties.broker().url());
        if (factory.getVirtualHost() == null || factory.getVirtualHost().isEmpty()) {
            factory.setVirtualHost(DEFAULT_VIRTUAL_HOST);
        }
        factory.setConnectionTimeout(CONNECTION_TIMEOUT_MS);
        factory.setRequestedHeartBeat(HEARTBEAT_SECONDS);
        factory.setPublisherConfirmType(CachingConnectionFactory.ConfirmType.CORRELATED);
        factory.setPublisherReturns(true);
        return factory;
    }

    @Bean
    public RabbitTemplate rabbitTemplate(CachingConnectionFactory connectionFactory) {
        RabbitTemplate template = new RabbitTemplate(connectionFactory);
        template.setMandatory(true);
        template.setReturnsCallback(returned -> log.warn(
                "message was not routed to any queue exchange={} routingKey={} replyText={}",
                returned.getExchange(), returned.getRoutingKey(), returned.getReplyText()));
        return template;
    }

    @Bean(LISTENER_FACTORY)
    public SimpleRabbitListenerContainerFactory paymentListenerFactory(
            CachingConnectionFactory connectionFactory, PaymentProperties properties) {
        SimpleRabbitListenerContainerFactory factory = new SimpleRabbitListenerContainerFactory();
        factory.setConnectionFactory(connectionFactory);
        factory.setAcknowledgeMode(AcknowledgeMode.MANUAL);
        factory.setPrefetchCount(properties.broker().prefetch());
        factory.setConcurrentConsumers(properties.broker().concurrency());
        factory.setMaxConcurrentConsumers(properties.broker().concurrency());
        factory.setMissingQueuesFatal(false);
        factory.setDefaultRequeueRejected(true);
        return factory;
    }
}
