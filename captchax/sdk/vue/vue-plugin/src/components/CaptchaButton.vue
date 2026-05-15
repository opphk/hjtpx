<template>
  <button 
    class="captcha-button"
    :class="[`captcha-button--${size}`, `captcha-button--${theme}`]"
    :disabled="disabled || loading"
    @click="handleClick"
  >
    <span v-if="loading" class="captcha-button__loader"></span>
    <slot>{{ text }}</slot>
  </button>
</template>

<script setup>
import { ref } from 'vue';
import { useCaptcha } from '../composables/useCaptcha';

const props = defineProps({
  scene: {
    type: String,
    default: 'default'
  },
  text: {
    type: String,
    default: '验证'
  },
  size: {
    type: String,
    default: 'medium',
    validator: (value) => ['small', 'medium', 'large'].includes(value)
  },
  theme: {
    type: String,
    default: 'light',
    validator: (value) => ['light', 'dark'].includes(value)
  },
  disabled: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['success', 'error']);

const { verify } = useCaptcha();
const loading = ref(false);

const handleClick = async () => {
  if (loading.value || props.disabled) {
    return;
  }
  
  loading.value = true;
  
  try {
    const token = await verify(props.scene);
    emit('success', token);
  } catch (error) {
    emit('error', error);
  } finally {
    loading.value = false;
  }
};
</script>

<style scoped>
.captcha-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  transition: all 0.2s ease;
}

.captcha-button--small {
  font-size: 12px;
  padding: 6px 12px;
}

.captcha-button--medium {
  font-size: 14px;
  padding: 8px 16px;
}

.captcha-button--large {
  font-size: 16px;
  padding: 12px 24px;
}

.captcha-button--light {
  background-color: #1890ff;
  color: #ffffff;
}

.captcha-button--light:hover:not(:disabled) {
  background-color: #40a9ff;
}

.captcha-button--dark {
  background-color: #001529;
  color: #ffffff;
}

.captcha-button--dark:hover:not(:disabled) {
  background-color: #1890ff;
}

.captcha-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.captcha-button__loader {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: captcha-button-spin 0.8s linear infinite;
}

@keyframes captcha-button-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
