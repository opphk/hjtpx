import { ref, readonly } from 'vue';

const state = {
  isVisible: ref(false),
  isLoading: ref(false),
  token: ref(null),
  error: ref(null)
};

export const useCaptchaState = () => {
  const show = () => {
    state.isVisible.value = true;
    state.error.value = null;
  };
  
  const hide = () => {
    state.isVisible.value = false;
  };
  
  const setLoading = (loading) => {
    state.isLoading.value = loading;
  };
  
  const setToken = (newToken) => {
    state.token.value = newToken;
  };
  
  const setError = (error) => {
    state.error.value = error;
  };
  
  const reset = () => {
    state.token.value = null;
    state.error.value = null;
    state.isLoading.value = false;
  };
  
  return {
    show,
    hide,
    setLoading,
    setToken,
    setError,
    reset,
    isVisible: readonly(state.isVisible),
    isLoading: readonly(state.isLoading),
    token: readonly(state.token),
    error: readonly(state.error)
  };
};
