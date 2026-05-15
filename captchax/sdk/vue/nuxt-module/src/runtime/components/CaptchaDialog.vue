<template>
  <Teleport to="body">
    <Transition name="captcha-dialog">
      <div v-if="visible" class="captcha-dialog-overlay" @click.self="handleClose">
        <div class="captcha-dialog-container">
          <div class="captcha-dialog-header">
            <h3 class="captcha-dialog-title">{{ title }}</h3>
            <button class="captcha-dialog-close" @click="handleClose" aria-label="关闭">
              <span>&times;</span>
            </button>
          </div>
          
          <div class="captcha-dialog-content">
            <slot>
              <component 
                :is="currentComponent" 
                :target-image="targetImage"
                :slider-image="sliderImage"
                @success="handleSuccess"
                @error="handleError"
              />
            </slot>
          </div>
          
          <div v-if="loading" class="captcha-dialog-loading">
            <div class="captcha-dialog-loader"></div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue';
import CaptchaSlider from './CaptchaSlider.vue';
import { useCaptcha } from '../composables/useCaptcha';

interface Props {
  visible: boolean;
  type?: 'slider' | 'click' | 'rotate' | 'puzzle' | 'text' | 'icon';
  title?: string;
  targetImage?: string;
  sliderImage?: string;
}

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  type: 'slider',
  title: '安全验证',
  targetImage: '',
  sliderImage: ''
});

const emit = defineEmits<{
  'update:visible': [visible: boolean];
  success: [token: string];
  error: [error: Error];
  close: [];
}>();

const { verify } = useCaptcha();
const loading = ref(false);

const currentComponent = computed(() => {
  switch (props.type) {
    case 'slider':
      return CaptchaSlider;
    default:
      return CaptchaSlider;
  }
});

watch(() => props.visible, (newVal) => {
  if (newVal) {
    loading.value = false;
  }
});

const handleClose = () => {
  emit('update:visible', false);
  emit('close');
};

const handleSuccess = async (data: string) => {
  loading.value = true;
  
  try {
    const token = await verify(props.type);
    emit('success', token);
    handleClose();
  } catch (error) {
    emit('error', error as Error);
  } finally {
    loading.value = false;
  }
};

const handleError = (error: Error) => {
  emit('error', error);
};
</script>

<style scoped>
.captcha-dialog-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 9999;
}

.captcha-dialog-container {
  background: white;
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  min-width: 320px;
  max-width: 90vw;
  position: relative;
}

.captcha-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #e8e8e8;
}

.captcha-dialog-title {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
  color: #333;
}

.captcha-dialog-close {
  background: none;
  border: none;
  font-size: 24px;
  line-height: 1;
  color: #999;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.captcha-dialog-close:hover {
  color: #666;
}

.captcha-dialog-content {
  padding: 16px;
}

.captcha-dialog-loading {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(255, 255, 255, 0.9);
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
}

.captcha-dialog-loader {
  width: 40px;
  height: 40px;
  border: 3px solid #e8e8e8;
  border-top-color: #1890ff;
  border-radius: 50%;
  animation: captcha-dialog-spin 0.8s linear infinite;
}

@keyframes captcha-dialog-spin {
  to {
    transform: rotate(360deg);
  }
}

.captcha-dialog-enter-active,
.captcha-dialog-leave-active {
  transition: opacity 0.3s ease;
}

.captcha-dialog-enter-from,
.captcha-dialog-leave-to {
  opacity: 0;
}

.captcha-dialog-enter-active .captcha-dialog-container,
.captcha-dialog-leave-active .captcha-dialog-container {
  transition: transform 0.3s ease;
}

.captcha-dialog-enter-from .captcha-dialog-container,
.captcha-dialog-leave-to .captcha-dialog-container {
  transform: scale(0.9);
}
</style>
