package com.captchax.sdk

enum class CaptchaType {
    SLIDER,
    CLICK,
    ROTATE,
    PUZZLE,
    TEXT,
    ICON;
    
    companion object {
        fun fromString(value: String): CaptchaType {
            return when (value.lowercase()) {
                "slider" -> SLIDER
                "click" -> CLICK
                "rotate" -> ROTATE
                "puzzle" -> PUZZLE
                "text" -> TEXT
                "icon" -> ICON
                else -> SLIDER
            }
        }
    }
}

data class CaptchaRequest(
    val scene: String,
    val type: CaptchaType = CaptchaType.SLIDER,
    val width: Int = 300,
    val height: Int = 200
)

data class CaptchaResponse(
    val token: String,
    val type: CaptchaType,
    val data: Map<String, Any>
)

data class CaptchaVerifyRequest(
    val token: String,
    val data: Map<String, Any>
)

data class CaptchaVerifyResponse(
    val success: Boolean,
    val token: String? = null,
    val errorCode: String? = null,
    val errorMessage: String? = null
)
