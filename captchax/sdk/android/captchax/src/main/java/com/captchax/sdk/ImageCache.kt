package com.captchax.sdk

import android.graphics.Bitmap
import android.util.LruCache
import com.captchax.sdk.util.Logger

class ImageCache(private val maxSize: Int = 50) {
    
    private val cache: LruCache<String, Bitmap>
    
    init {
        val maxMemory = (Runtime.getRuntime().maxMemory() / 1024).toInt()
        val cacheSize = maxMemory / 8
        
        val actualSize = if (maxSize > 0) maxSize else cacheSize
        
        cache = object : LruCache<String, Bitmap>(actualSize) {
            override fun sizeOf(key: String, bitmap: Bitmap): Int {
                return bitmap.byteCount / 1024
            }
            
            override fun entryRemoved(
                evicted: Boolean,
                key: String,
                oldValue: Bitmap,
                newValue: Bitmap?
            ) {
                if (evicted && !oldValue.isRecycled) {
                    Logger.d("ImageCache", "Entry removed: $key")
                }
            }
        }
    }
    
    fun put(key: String, bitmap: Bitmap) {
        if (get(key) == null) {
            cache.put(key, bitmap)
            Logger.d("ImageCache", "Cached: $key")
        }
    }
    
    fun get(key: String): Bitmap? {
        val bitmap = cache.get(key)
        if (bitmap != null && !bitmap.isRecycled) {
            Logger.d("ImageCache", "Cache hit: $key")
            return bitmap
        }
        Logger.d("ImageCache", "Cache miss: $key")
        return null
    }
    
    fun remove(key: String) {
        cache.remove(key)
        Logger.d("ImageCache", "Removed: $key")
    }
    
    fun clear() {
        val snapshots = cache.snapshot()
        cache.evictAll()
        
        snapshots.values.forEach { bitmap ->
            if (!bitmap.isRecycled) {
                bitmap.recycle()
            }
        }
        
        Logger.d("ImageCache", "Cache cleared")
    }
    
    fun size(): Int {
        return cache.size()
    }
    
    fun maxSize(): Int {
        return cache.maxSize()
    }
    
    fun hitCount(): Int {
        return cache.hitCount()
    }
    
    fun missCount(): Int {
        return cache.missCount()
    }
    
    fun evictionCount(): Int {
        return cache.evictionCount()
    }
}
