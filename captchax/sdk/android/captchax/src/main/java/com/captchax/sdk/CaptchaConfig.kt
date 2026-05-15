package com.captchax.sdk

data class CaptchaConfig(
    val apiKey: String,
    val apiSecret: String,
    val serverUrl: String,
    val timeout: Long = 30000L,
    val cacheEnabled: Boolean = true,
    val preloadEnabled: Boolean = true
) {
    init {
        require(apiKey.isNotBlank()) { "API Key cannot be blank" }
        require(apiSecret.isNotBlank()) { "API Secret cannot be blank" }
        require(serverUrl.isNotBlank()) { "Server URL cannot be blank" }
        require(timeout > 0) { "Timeout must be positive" }
    }

    companion object {
        fun builder() = Builder()

        class Builder {
            private var apiKey: String = ""
            private var apiSecret: String = ""
            private var serverUrl: String = "https://api.captchax.com"
            private var timeout: Long = 30000L
            private var cacheEnabled: Boolean = true
            private var preloadEnabled: Boolean = true

            fun apiKey(key: String) = apply { this.apiKey = key }
            fun apiSecret(secret: String) = apply { this.apiSecret = secret }
            fun serverUrl(url: String) = apply { this.serverUrl = url }
            fun timeout(ms: Long) = apply { this.timeout = ms }
            fun cacheEnabled(enabled: Boolean) = apply { this.cacheEnabled = enabled }
            fun preloadEnabled(enabled: Boolean) = apply { this.preloadEnabled = enabled }

            fun build(): CaptchaConfig {
                return CaptchaConfig(
                    apiKey = apiKey,
                    apiSecret = apiSecret,
                    serverUrl = serverUrl,
                    timeout = timeout,
                    cacheEnabled = cacheEnabled,
                    preloadEnabled = preloadEnabled
                )
            }
        }
    }
}
