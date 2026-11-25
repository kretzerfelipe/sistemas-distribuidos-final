package com.futhub.payment.web;

import java.util.LinkedHashMap;
import java.util.Map;

import org.springframework.amqp.rabbit.connection.CachingConnectionFactory;
import org.springframework.amqp.rabbit.connection.Connection;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class HealthController {

    private static final String UP = "up";
    private static final String DOWN = "down";

    private final JdbcTemplate jdbc;
    private final CachingConnectionFactory connectionFactory;

    public HealthController(JdbcTemplate jdbc, CachingConnectionFactory connectionFactory) {
        this.jdbc = jdbc;
        this.connectionFactory = connectionFactory;
    }

    @GetMapping("/health")
    public ResponseEntity<Map<String, Object>> health() {
        boolean databaseUp = isDatabaseUp();
        boolean brokerUp = isBrokerUp();

        Map<String, Object> checks = new LinkedHashMap<>();
        checks.put("database", databaseUp ? UP : DOWN);
        checks.put("rabbitmq", brokerUp ? UP : DOWN);

        Map<String, Object> body = new LinkedHashMap<>();
        body.put("status", databaseUp && brokerUp ? "ok" : "degraded");
        body.put("checks", checks);

        HttpStatus status = databaseUp ? HttpStatus.OK : HttpStatus.SERVICE_UNAVAILABLE;
        return ResponseEntity.status(status).body(body);
    }

    private boolean isDatabaseUp() {
        try {
            jdbc.queryForObject("SELECT 1", Integer.class);
            return true;
        } catch (RuntimeException e) {
            return false;
        }
    }

    private boolean isBrokerUp() {
        try {
            Connection connection = connectionFactory.createConnection();
            return connection.isOpen();
        } catch (RuntimeException e) {
            return false;
        }
    }
}
