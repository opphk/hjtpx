package com.hjtpx.sdk;

public class PoolStats {
    private int activeConnections;
    private int idleConnections;
    private long totalRequests;
    private long failedRequests;
    private long successfulRequests;
    private long retriedRequests;
    private double successRate;
    private String lastError;
    private String lastErrorTime;

    public PoolStats() {}

    public int getActiveConnections() {
        return activeConnections;
    }

    public void setActiveConnections(int activeConnections) {
        this.activeConnections = activeConnections;
    }

    public int getIdleConnections() {
        return idleConnections;
    }

    public void setIdleConnections(int idleConnections) {
        this.idleConnections = idleConnections;
    }

    public long getTotalRequests() {
        return totalRequests;
    }

    public void setTotalRequests(long totalRequests) {
        this.totalRequests = totalRequests;
    }

    public long getFailedRequests() {
        return failedRequests;
    }

    public void setFailedRequests(long failedRequests) {
        this.failedRequests = failedRequests;
    }

    public long getSuccessfulRequests() {
        return successfulRequests;
    }

    public void setSuccessfulRequests(long successfulRequests) {
        this.successfulRequests = successfulRequests;
    }

    public long getRetriedRequests() {
        return retriedRequests;
    }

    public void setRetriedRequests(long retriedRequests) {
        this.retriedRequests = retriedRequests;
    }

    public double getSuccessRate() {
        return successRate;
    }

    public void setSuccessRate(double successRate) {
        this.successRate = successRate;
    }

    public String getLastError() {
        return lastError;
    }

    public void setLastError(String lastError) {
        this.lastError = lastError;
    }

    public String getLastErrorTime() {
        return lastErrorTime;
    }

    public void setLastErrorTime(String lastErrorTime) {
        this.lastErrorTime = lastErrorTime;
    }

    @Override
    public String toString() {
        return "PoolStats{" +
                "activeConnections=" + activeConnections +
                ", idleConnections=" + idleConnections +
                ", totalRequests=" + totalRequests +
                ", failedRequests=" + failedRequests +
                ", successfulRequests=" + successfulRequests +
                ", retriedRequests=" + retriedRequests +
                ", successRate=" + successRate +
                ", lastError='" + lastError + '\'' +
                ", lastErrorTime='" + lastErrorTime + '\'' +
                '}';
    }
}
