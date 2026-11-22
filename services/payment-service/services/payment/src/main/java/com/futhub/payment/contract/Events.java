package com.futhub.payment.contract;

public final class Events {

    public static final String PAYMENT_REQUESTED = "PaymentRequested";
    public static final String PAYMENT_APPROVED = "PaymentApproved";
    public static final String PAYMENT_REJECTED = "PaymentRejected";
    public static final String SUBSCRIPTION_CANCELED = "SubscriptionCanceled";
    public static final String REFUND_REQUESTED = "RefundRequested";
    public static final String REFUND_COMPLETED = "RefundCompleted";

    public static final String PAYMENT_APPROVED_KEY = "payment.approved";
    public static final String PAYMENT_REJECTED_KEY = "payment.rejected";
    public static final String REFUND_COMPLETED_KEY = "refund.completed";

    private Events() {
    }
}
