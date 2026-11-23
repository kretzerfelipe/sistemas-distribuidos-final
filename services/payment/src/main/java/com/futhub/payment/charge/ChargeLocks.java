package com.futhub.payment.charge;

import java.util.Optional;
import java.util.UUID;

public interface ChargeLocks {

    Optional<Lock> tryAcquire(UUID subscriptionId);

    interface Lock extends AutoCloseable {

        @Override
        void close();
    }
}
