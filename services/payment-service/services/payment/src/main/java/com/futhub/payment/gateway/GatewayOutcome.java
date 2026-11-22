package com.futhub.payment.gateway;

public enum GatewayOutcome {
    APPROVED(null),
    DECLINED("GATEWAY_DECLINED"),
    TIMEOUT("GATEWAY_TIMEOUT"),
    ERROR("GATEWAY_ERROR");

    private final String reason;

    GatewayOutcome(String reason) {
        this.reason = reason;
    }

    public String reason() {
        return reason;
    }

    public boolean isApproved() {
        return this == APPROVED;
    }
}
