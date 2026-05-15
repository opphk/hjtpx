package com.captchax.sdk.util

import android.content.Context
import android.view.View
import android.view.inputmethod.InputMethodManager
import android.widget.Toast
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.TimeoutCancellationException
import kotlinx.coroutines.withTimeoutOrNull

fun Context.showToast(message: String, duration: Int = Toast.LENGTH_SHORT) {
    Toast.makeText(this, message, duration).show()
}

fun Context.hideKeyboard(view: View) {
    val imm = getSystemService(Context.INPUT_METHOD_SERVICE) as InputMethodManager
    imm.hideSoftInputFromWindow(view.windowToken, 0)
}

fun Context.showKeyboard(view: View) {
    view.requestFocus()
    val imm = getSystemService(Context.INPUT_METHOD_SERVICE) as InputMethodManager
    imm.showSoftInput(view, InputMethodManager.SHOW_IMPLICIT)
}

fun View.visible() {
    visibility = View.VISIBLE
}

fun View.gone() {
    visibility = View.GONE
}

fun View.invisible() {
    visibility = View.INVISIBLE
}

fun View.visibleIf(condition: Boolean) {
    visibility = if (condition) View.VISIBLE else View.GONE
}

inline fun <T> Result<T>.onSuccessFinally(crossinline action: (T) -> Unit): Result<T> {
    if (isSuccess) {
        getOrNull()?.let { action(it) }
    }
    return this
}

inline fun <T> Result<T>.onFailureFinally(crossinline action: (Throwable) -> Unit): Result<T> {
    if (isFailure) {
        exceptionOrNull()?.let { action(it) }
    }
    return this
}

suspend fun <T> withTimeoutResult(
    timeMillis: Long,
    default: T,
    block: suspend () -> T
): Result<T> {
    return try {
        val result = withTimeoutOrNull(timeMillis, block) ?: default
        Result.success(result)
    } catch (e: CancellationException) {
        Result.failure(e)
    } catch (e: TimeoutCancellationException) {
        Result.failure(e)
    } catch (e: Exception) {
        Result.failure(e)
    }
}

fun String.isValidUrl(): Boolean {
    return try {
        val url = java.net.URL(this)
        url.protocol in listOf("http", "https")
    } catch (e: Exception) {
        false
    }
}

fun String.md5(): String {
    val md = java.security.MessageDigest.getInstance("MD5")
    val digest = md.digest(this.toByteArray())
    return digest.joinToString("") { "%02x".format(it) }
}

fun String.sha256(): String {
    val md = java.security.MessageDigest.getInstance("SHA-256")
    val digest = md.digest(this.toByteArray())
    return digest.joinToString("") { "%02x".format(it) }
}

fun ByteArray.toHexString(): String {
    return joinToString("") { "%02x".format(it) }
}

fun Long.toReadableSize(): String {
    if (this < 1024) return "$this B"
    val kb = this / 1024.0
    if (kb < 1024) return "%.2f KB".format(kb)
    val mb = kb / 1024.0
    if (mb < 1024) return "%.2f MB".format(mb)
    val gb = mb / 1024.0
    return "%.2f GB".format(gb)
}

fun Int.toDp(context: Context): Int {
    return (this * context.resources.displayMetrics.density).toInt()
}

fun Int.dpToPx(context: Context): Int {
    return (this * context.resources.displayMetrics.density).toInt()
}

fun Float.toDp(context: Context): Float {
    return this * context.resources.displayMetrics.density
}

fun Float.dpToPx(context: Context): Float {
    return this * context.resources.displayMetrics.density
}
