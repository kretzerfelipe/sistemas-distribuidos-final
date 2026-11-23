package com.futhub.payment.web;

import java.util.Map;

import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;

final class ApiErrors {

    private ApiErrors() {
    }

    static ResponseEntity<Map<String, Object>> of(HttpStatus status, String code, String message) {
        return ResponseEntity.status(status).body(Map.of("error", Map.of("code", code, "message", message)));
    }
}
