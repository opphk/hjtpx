import React from 'react';
import {
  View,
  Image,
  StyleSheet,
  TouchableOpacity,
  Text,
} from 'react-native';

interface CaptchaButtonProps {
  onPress: () => void;
  title?: string;
  disabled?: boolean;
  loading?: boolean;
}

export const CaptchaButton: React.FC<CaptchaButtonProps> = ({
  onPress,
  title = '验证',
  disabled = false,
  loading = false,
}) => {
  return (
    <TouchableOpacity
      style={[styles.button, disabled && styles.buttonDisabled]}
      onPress={onPress}
      disabled={disabled || loading}
      activeOpacity={0.7}
    >
      <Text style={[styles.buttonText, disabled && styles.buttonTextDisabled]}>
        {loading ? '加载中...' : title}
      </Text>
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  button: {
    backgroundColor: '#1890ff',
    paddingVertical: 12,
    paddingHorizontal: 24,
    borderRadius: 4,
    alignItems: 'center',
    justifyContent: 'center',
  },
  buttonDisabled: {
    backgroundColor: '#d9d9d9',
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '500',
  },
  buttonTextDisabled: {
    color: '#bfbfbf',
  },
});

export default CaptchaButton;
