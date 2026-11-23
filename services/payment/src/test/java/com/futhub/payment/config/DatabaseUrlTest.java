package com.futhub.payment.config;

import static org.assertj.core.api.Assertions.assertThat;

import org.junit.jupiter.api.Test;

class DatabaseUrlTest {

    private static final String URL = "postgres://futhub:futhub123@ministack-rds-1-instance-futhub-payment:5432/db_payment";

    @Test
    void convertsAPostgresUrlToJdbc() {
        assertThat(DatabaseUrl.toJdbc(URL))
                .isEqualTo("jdbc:postgresql://ministack-rds-1-instance-futhub-payment:5432/db_payment");
    }

    @Test
    void extractsTheCredentials() {
        DatabaseUrl.Credentials credentials = DatabaseUrl.credentials(URL);

        assertThat(credentials.user()).isEqualTo("futhub");
        assertThat(credentials.password()).isEqualTo("futhub123");
    }

    @Test
    void usesTheDefaultPortAndKeepsTheQuery() {
        assertThat(DatabaseUrl.toJdbc("postgres://u:p@localhost/db_payment?sslmode=disable"))
                .isEqualTo("jdbc:postgresql://localhost:5432/db_payment?sslmode=disable");
    }

    @Test
    void keepsAJdbcUrlAsItIs() {
        assertThat(DatabaseUrl.toJdbc("jdbc:postgresql://localhost:5432/db")).isEqualTo("jdbc:postgresql://localhost:5432/db");
    }
}
