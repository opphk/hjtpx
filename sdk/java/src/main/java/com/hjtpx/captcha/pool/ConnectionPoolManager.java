package com.hjtpx.captcha.pool;

import org.apache.http.client.config.RequestConfig;
import org.apache.http.config.Registry;
import org.apache.http.config.RegistryBuilder;
import org.apache.http.conn.socket.ConnectionSocketFactory;
import org.apache.http.conn.socket.PlainConnectionSocketFactory;
import org.apache.http.conn.ssl.SSLConnectionSocketFactory;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.impl.conn.PoolingHttpClientConnectionManager;

import java.util.concurrent.TimeUnit;

public class ConnectionPoolManager implements AutoCloseable {
    private final PoolingHttpClientConnectionManager connectionManager;
    private final CloseableHttpClient httpClient;
    private final ConnectionPoolConfig config;

    public ConnectionPoolManager() {
        this(new ConnectionPoolConfig());
    }

    public ConnectionPoolManager(ConnectionPoolConfig config) {
        this.config = config;

        Registry<ConnectionSocketFactory> socketFactoryRegistry = RegistryBuilder.<ConnectionSocketFactory>create()
            .register("http", PlainConnectionSocketFactory.getSocketFactory())
            .register("https", SSLConnectionSocketFactory.getSocketFactory())
            .build();

        connectionManager = new PoolingHttpClientConnectionManager(socketFactoryRegistry, null, null, null, config.getTimeToLive(), TimeUnit.MILLISECONDS);
        connectionManager.setMaxTotal(config.getMaxConnections());
        connectionManager.setDefaultMaxPerRoute(config.getMaxConnectionsPerRoute());

        if (config.isValidateAfterInactivity()) {
            connectionManager.setValidateAfterInactivity(config.getValidateAfterInactivityMillis());
        }

        RequestConfig requestConfig = RequestConfig.custom()
            .setConnectTimeout(config.getConnectionTimeout())
            .setSocketTimeout(config.getSocketTimeout())
            .setConnectionRequestTimeout(config.getConnectionRequestTimeout())
            .build();

        httpClient = HttpClients.custom()
            .setConnectionManager(connectionManager)
            .setDefaultRequestConfig(requestConfig)
            .evictIdleConnections(config.getTimeToLive(), TimeUnit.MILLISECONDS)
            .evictExpiredConnections()
            .build();
    }

    public CloseableHttpClient getHttpClient() {
        return httpClient;
    }

    public PoolingHttpClientConnectionManager getConnectionManager() {
        return connectionManager;
    }

    public ConnectionPoolConfig getConfig() {
        return config;
    }

    @Override
    public void close() throws Exception {
        if (httpClient != null) {
            httpClient.close();
        }
    }
}
