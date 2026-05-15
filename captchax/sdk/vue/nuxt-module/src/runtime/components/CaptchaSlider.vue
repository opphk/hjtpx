<template>
  <div class="captcha-slider">
    <div class="captcha-slider__background" :style="backgroundStyle">
      <div 
        class="captcha-slider__target" 
        :style="targetStyle"
      ></div>
    </div>
    
    <div class="captcha-slider__track" ref="trackRef">
      <div 
        class="captcha-slider__thumb" 
        :style="thumbStyle"
        @mousedown="handleDragStart"
        @touchstart.passive="handleDragStart"
      >
        <svg class="captcha-slider__arrow" viewBox="0 0 24 24" fill="currentColor">
          <path d="M8.59 16.59L13.17 12 8.59 7.41 10 6l6 6-6 6-1.41-1.41z"/>
        </svg>
      </div>
    </div>
    
    <div class="captcha-slider__tips">
      <span v-if="!isVerified">{{ tips }}</span>
      <span v-else class="captcha-slider__success">验证成功</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';
import { useCaptchaState } from '../composables/useCaptchaState';

interface Props {
  targetImage?: string;
  sliderImage?: string;
}

const props = withDefaults(defineProps<Props>(), {
  targetImage: '',
  sliderImage: ''
});

const emit = defineEmits<{
  success: [token: string];
  error: [error: Error];
}>();

const { setLoading, setToken, setError, isLoading } = useCaptchaState();

const isDragging = ref(false);
const isVerified = ref(false);
const distance = ref(0);
const trackRef = ref<HTMLElement | null>(null);
const tips = ref('拖动滑块完成拼图');

const targetPosition = Math.floor(Math.random() * 40) + 30;

const backgroundStyle = computed(() => ({
  backgroundImage: props.targetImage ? `url(${props.targetImage})` : 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  backgroundSize: 'cover'
}));

const targetStyle = computed(() => ({
  left: `${targetPosition}%`,
  top: `${Math.floor(Math.random() * 40) + 10}%`
}));

const thumbStyle = computed(() => ({
  transform: `translateX(${distance.value}px)`
}));

let startX = 0;

const handleDragStart = (e: MouseEvent | TouchEvent) => {
  if (isVerified.value || isLoading.value) return;
  
  isDragging.value = true;
  startX = 'touches' in e ? e.touches[0].clientX : e.clientX;
  
  document.addEventListener('mousemove', handleDragMove);
  document.addEventListener('mouseup', handleDragEnd);
  document.addEventListener('touchmove', handleDragMove);
  document.addEventListener('touchend', handleDragEnd);
};

const handleDragMove = (e: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return;
  
  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
  const deltaX = clientX - startX;
  
  if (trackRef.value) {
    const maxDistance = trackRef.value.offsetWidth - 40;
    distance.value = Math.max(0, Math.min(deltaX, maxDistance));
  }
};

const handleDragEnd = async () => {
  if (!isDragging.value) return;
  
  isDragging.value = false;
  
  document.removeEventListener('mousemove', handleDragMove);
  document.removeEventListener('mouseup', handleDragEnd);
  document.removeEventListener('touchmove', handleDragMove);
  document.removeEventListener('touchend', handleDragEnd);
  
  const threshold = targetPosition * (trackRef.value?.offsetWidth / 100);
  const tolerance = 10;
  
  if (Math.abs(distance.value - threshold) <= tolerance) {
    isVerified.value = true;
    tips.value = '';
    
    try {
      const token = `slider_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
      setToken(token);
      setLoading(false);
      emit('success', token);
    } catch (error) {
      setError(error as Error);
      emit('error', error as Error);
    }
  } else {
    distance.value = 0;
    tips.value = '验证失败，请重试';
    emit('error', new Error('Verification failed'));
  }
};
</script>

<style scoped>
.captcha-slider {
  width: 300px;
  user-select: none;
}

.captcha-slider__background {
  position: relative;
  width: 100%;
  height: 150px;
  border-radius: 4px;
  overflow: hidden;
}

.captcha-slider__target {
  position: absolute;
  width: 40px;
  height: 40px;
  background: rgba(255, 255, 255, 0.9);
  border-radius: 4px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.captcha-slider__track {
  position: relative;
  width: 100%;
  height: 40px;
  background: #f0f0f0;
  border-radius: 20px;
  margin-top: 16px;
  overflow: hidden;
}

.captcha-slider__thumb {
  position: absolute;
  left: 0;
  top: 0;
  width: 40px;
  height: 40px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  cursor: grab;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
  transition: transform 0.1s ease;
}

.captcha-slider__thumb:hover {
  transform: scale(1.05);
}

.captcha-slider__thumb:active {
  cursor: grabbing;
}

.captcha-slider__arrow {
  width: 20px;
  height: 20px;
  color: white;
}

.captcha-slider__tips {
  text-align: center;
  margin-top: 12px;
  font-size: 14px;
  color: #666;
}

.captcha-slider__success {
  color: #52c41a;
}
</style>
