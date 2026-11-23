package com.futhub.payment.charge;

import java.sql.Connection;
import java.sql.PreparedStatement;
import java.sql.ResultSet;
import java.sql.SQLException;
import java.util.Optional;
import java.util.UUID;
import javax.sql.DataSource;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.stereotype.Component;

@Component
public class AdvisoryChargeLocks implements ChargeLocks {

    private static final Logger log = LoggerFactory.getLogger(AdvisoryChargeLocks.class);

    private final DataSource dataSource;

    public AdvisoryChargeLocks(DataSource dataSource) {
        this.dataSource = dataSource;
    }

    @Override
    public Optional<Lock> tryAcquire(UUID subscriptionId) {
        long key = subscriptionId.getMostSignificantBits() ^ subscriptionId.getLeastSignificantBits();
        Connection connection = null;
        try {
            connection = dataSource.getConnection();
            if (!tryLock(connection, key)) {
                connection.close();
                return Optional.empty();
            }
            Connection held = connection;
            return Optional.of(() -> release(held, key));
        } catch (SQLException e) {
            closeQuietly(connection);
            throw new IllegalStateException("could not acquire the lock of subscription " + subscriptionId, e);
        }
    }

    private boolean tryLock(Connection connection, long key) throws SQLException {
        try (PreparedStatement statement = connection.prepareStatement("SELECT pg_try_advisory_lock(?)")) {
            statement.setLong(1, key);
            try (ResultSet result = statement.executeQuery()) {
                return result.next() && result.getBoolean(1);
            }
        }
    }

    private void release(Connection connection, long key) {
        try (PreparedStatement statement = connection.prepareStatement("SELECT pg_advisory_unlock(?)")) {
            statement.setLong(1, key);
            statement.execute();
            connection.close();
        } catch (SQLException e) {
            log.warn("could not release the charge lock, dropping the connection", e);
            abortQuietly(connection);
        }
    }

    private void abortQuietly(Connection connection) {
        try {
            connection.abort(Runnable::run);
        } catch (SQLException | RuntimeException e) {
            closeQuietly(connection);
        }
    }

    private void closeQuietly(Connection connection) {
        if (connection == null) {
            return;
        }
        try {
            connection.close();
        } catch (SQLException ignored) {
            return;
        }
    }
}
