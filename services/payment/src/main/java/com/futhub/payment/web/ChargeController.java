package com.futhub.payment.web;

import java.util.LinkedHashMap;
import java.util.Map;
import java.util.Optional;
import java.util.UUID;

import com.futhub.payment.charge.Charge;
import com.futhub.payment.charge.ChargeRepository;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class ChargeController {

    private final ChargeRepository charges;

    public ChargeController(ChargeRepository charges) {
        this.charges = charges;
    }

    @GetMapping("/charges/{id}")
    public ResponseEntity<?> get(@PathVariable String id) {
        UUID chargeId;
        try {
            chargeId = UUID.fromString(id);
        } catch (IllegalArgumentException e) {
            return ApiErrors.of(HttpStatus.UNPROCESSABLE_ENTITY, "invalid_input", "id must be a UUID");
        }

        Optional<Charge> charge = charges.findById(chargeId);
        if (charge.isEmpty()) {
            return ApiErrors.of(HttpStatus.NOT_FOUND, "not_found", "charge not found");
        }
        return ResponseEntity.ok(toResponse(charge.get()));
    }

    private Map<String, Object> toResponse(Charge charge) {
        Map<String, Object> body = new LinkedHashMap<>();
        body.put("id", charge.id());
        body.put("subscriptionId", charge.subscriptionId());
        body.put("userId", charge.userId());
        body.put("amountCents", charge.amountCents());
        body.put("status", charge.status());
        body.put("attempts", charge.attempts());
        body.put("refundPending", charge.refundPending());
        body.put("failureReason", charge.failureReason());
        body.put("createdAt", charge.createdAt().toString());
        body.put("updatedAt", charge.updatedAt().toString());
        return body;
    }
}
