package com.hjtpx.captcha.pool;

public class ConnectionPoolConfig {
    private int maxConnections = 100;
    private int maxConnectionsPerRoute = 50;
    private int connectionTimeout = 5000;
    private int socketTimeout = 30000;
    private int connectionRequestTimeout = 5000;
    private int timeToLive = 60000;
    private boolean validateAfterInactivity = true;
    private int validateAfterInactivityMillis = 2000;

    public int getMaxConnections() {
        return maxConnections;
    }

    public void setMaxConnections(int maxConnections) {
        this.maxConnections = maxConnections;
    }

    public int getMaxConnectionsPerRoute() {
        return maxConnectionsPerRoute;
    }

    public void setMaxConnectionsPerRoute(int maxConnectionsPerRoute) {
        this.maxConnectionsPerRoute = maxConnectionsPerRoute;
    }

    public int getConnectionTimeout() {
        return connectionTimeout;
    }

    public void setConnectionTimeout(int connectionTimeout) {
        this.connectionTimeout = connectionTimeout;
    }

    public int getSocketTimeout() {
        return socketTimeout;
    }

    public void setSocketTimeout(int socketTimeout) {
        this.socketTimeout = socketTimeout;
    }

    public int getConnectionRequestTimeout() {
        return connectionRequestTimeout;
    }

    public void setConnectionRequestTimeout(int connectionRequestTimeout) {
        this.connectionRequestTimeout = connectionRequestTimeout;
    }

    public int getTimeToLive() {
        return timeToLive;
    }

    public void setTimeToLive(int timeToLive) {
        this.timeToLive = timeToLive;
    }

    public boolean isValidateAfterInactivity() {
        return validateAfterInactivity;
    }

    public void setValidateAfterInactivity(boolean validateAfterInactivity) {
        this.validateAfterInactivity = validateAfterInactivity;
    }

    public int getValidateAfterInactivityMillis() {
        return validateAfterInactivityMillis;
    }

    public void setValidateAfterInactivityMillis(int validateAfterInactivityMillis) {
        this.validateAfterInactivityMillis = validateAfterInactivityMillis;
    }
}
