package com.futhub.payment.messaging;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

import java.nio.charset.StandardCharsets;
import java.time.Instant;
import java.util.UUID;

import com.fasterxml.jackson.databind.ObjectMapper;
import com.futhub.payment.config.PaymentProperties;
import com.futhub.payment.contract.ContractValidator;
import com.futhub.payment.contract.Envelope;
import com.futhub.payment.contract.InvalidMessageException;
import org.junit.jupiter.api.Test;

class ContractValidatorTest {

    private static final String SUBSCRIPTION_ID = UUID.randomUUID().toString();
    private static final String USER_ID = UUID.randomUUID().toString();
    private static final String CHARGE_ID = UUID.randomUUID().toString();

    private final ContractValidator validator = new ContractValidator(
            new PaymentProperties("", "../../contracts", null, null, null), new ObjectMapper());

    @Test
    void acceptsAValidPaymentRequested() {
        String body = envelope("PaymentRequested",
                "{\"subscriptionId\":\"%s\",\"userId\":\"%s\",\"amountCents\":1990}".formatted(SUBSCRIPTION_ID, USER_ID));

        Envelope envelope = validator.validate(bytes(body));

        assertThat(envelope.type()).isEqualTo("PaymentRequested");
        assertThat(envelope.payload().get("amountCents").asInt()).isEqualTo(1990);
    }

    @Test
    void acceptsAValidRefundRequested() {
        String body = envelope("RefundRequested",
                "{\"subscriptionId\":\"%s\",\"chargeId\":\"%s\"}".formatted(SUBSCRIPTION_ID, CHARGE_ID));

        assertThat(validator.validate(bytes(body)).type()).isEqualTo("RefundRequested");
    }

    @Test
    void acceptsAValidSubscriptionCanceled() {
        String body = envelope("SubscriptionCanceled",
                "{\"subscriptionId\":\"%s\",\"userId\":\"%s\",\"reason\":\"TIMEOUT\"}".formatted(SUBSCRIPTION_ID, USER_ID));

        assertThat(validator.validate(bytes(body)).type()).isEqualTo("SubscriptionCanceled");
    }

    @Test
    void rejectsAPayloadWithAMissingField() {
        String body = envelope("PaymentRequested",
                "{\"subscriptionId\":\"%s\",\"userId\":\"%s\"}".formatted(SUBSCRIPTION_ID, USER_ID));

        assertThatThrownBy(() -> validator.validate(bytes(body))).isInstanceOf(InvalidMessageException.class);
    }

    @Test
    void rejectsAPayloadWithAnInvalidUuid() {
        String body = envelope("PaymentRequested",
                "{\"subscriptionId\":\"not-a-uuid\",\"userId\":\"%s\",\"amountCents\":1990}".formatted(USER_ID));

        assertThatThrownBy(() -> validator.validate(bytes(body))).isInstanceOf(InvalidMessageException.class);
    }

    @Test
    void rejectsAnUnknownCancellationReason() {
        String body = envelope("SubscriptionCanceled",
                "{\"subscriptionId\":\"%s\",\"userId\":\"%s\",\"reason\":\"BORED\"}".formatted(SUBSCRIPTION_ID, USER_ID));

        assertThatThrownBy(() -> validator.validate(bytes(body))).isInstanceOf(InvalidMessageException.class);
    }

    @Test
    void rejectsAnUnknownMessageType() {
        String body = envelope("SomethingElse", "{}");

        assertThatThrownBy(() -> validator.validate(bytes(body))).isInstanceOf(InvalidMessageException.class);
    }

    @Test
    void rejectsABodyThatIsNotJson() {
        assertThatThrownBy(() -> validator.validate(bytes("{not json"))).isInstanceOf(InvalidMessageException.class);
    }

    private static String envelope(String type, String payload) {
        return """
                {"messageId":"%s","correlationId":"%s","type":"%s","version":1,"occurredAt":"%s","payload":%s}"""
                .formatted(UUID.randomUUID(), UUID.randomUUID(), type, Instant.now(), payload);
    }

    private static byte[] bytes(String body) {
        return body.getBytes(StandardCharsets.UTF_8);
    }
}
