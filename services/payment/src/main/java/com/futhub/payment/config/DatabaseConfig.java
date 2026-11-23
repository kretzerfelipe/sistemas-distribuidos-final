package com.futhub.payment.config;

import javax.sql.DataSource;

import com.zaxxer.hikari.HikariConfig;
import com.zaxxer.hikari.HikariDataSource;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class DatabaseConfig {

    private static final int MAX_POOL_SIZE = 10;
    private static final long CONNECTION_TIMEOUT_MS = 5000;

    @Bean
    public DataSource dataSource(PaymentProperties properties) {
        HikariConfig config = new HikariConfig();
        config.setJdbcUrl(DatabaseUrl.toJdbc(properties.databaseUrl()));
        DatabaseUrl.Credentials credentials = DatabaseUrl.credentials(properties.databaseUrl());
        if (credentials.user() != null) {
            config.setUsername(credentials.user());
        }
        if (credentials.password() != null) {
            config.setPassword(credentials.password());
        }
        config.setMaximumPoolSize(MAX_POOL_SIZE);
        config.setConnectionTimeout(CONNECTION_TIMEOUT_MS);
        config.setPoolName("payment-db");
        return new HikariDataSource(config);
    }
}
