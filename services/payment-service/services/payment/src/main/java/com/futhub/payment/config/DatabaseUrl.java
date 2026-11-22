package com.futhub.payment.config;

import java.net.URI;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;

public final class DatabaseUrl {

    private static final String JDBC_PREFIX = "jdbc:";
    private static final int DEFAULT_POSTGRES_PORT = 5432;

    private DatabaseUrl() {
    }

    public record Credentials(String user, String password) {
    }

    public static String toJdbc(String url) {
        if (url.startsWith(JDBC_PREFIX)) {
            return url;
        }
        URI uri = URI.create(url);
        int port = uri.getPort() == -1 ? DEFAULT_POSTGRES_PORT : uri.getPort();
        String query = uri.getRawQuery() == null ? "" : "?" + uri.getRawQuery();
        return JDBC_PREFIX + "postgresql://" + uri.getHost() + ":" + port + uri.getRawPath() + query;
    }

    public static Credentials credentials(String url) {
        if (url.startsWith(JDBC_PREFIX)) {
            return new Credentials(null, null);
        }
        String userInfo = URI.create(url).getRawUserInfo();
        if (userInfo == null) {
            return new Credentials(null, null);
        }
        int separator = userInfo.indexOf(':');
        if (separator < 0) {
            return new Credentials(decode(userInfo), null);
        }
        return new Credentials(decode(userInfo.substring(0, separator)), decode(userInfo.substring(separator + 1)));
    }

    private static String decode(String value) {
        return URLDecoder.decode(value, StandardCharsets.UTF_8);
    }
}
