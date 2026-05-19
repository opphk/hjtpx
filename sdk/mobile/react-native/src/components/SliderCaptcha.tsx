import React from 'react';
import {
  View,
  Image,
  StyleSheet,
  PanResponder,
  Animated,
  GestureResponderEvent,
  PanResponderGestureState,
} from 'react-native';

interface SliderCaptchaProps {
  backgroundImageUrl: string;
  sliderImageUrl: string;
  onSliderMoved?: (progress: number) => void;
  onSliderCompleted: (progress: number) => void;
  width?: number;
  height?: number;
  trackHeight?: number;
  thumbSize?: number;
}

export const SliderCaptcha: React.FC<SliderCaptchaProps> = ({
  backgroundImageUrl,
  sliderImageUrl,
  onSliderMoved,
  onSliderCompleted,
  width = 320,
  height = 200,
  trackHeight = 4,
  thumbSize = 50,
}) => {
  const [sliderPosition] = React.useState(new Animated.Value(0));
  const [currentPosition, setCurrentPosition] = React.useState(0);
  const [isDragging, setIsDragging] = React.useState(false);
  const maxPosition = width - thumbSize - 10;

  const panResponder = React.useMemo(
    () =>
      PanResponder.create({
        onStartShouldSetPanResponder: () => true,
        onMoveShouldSetPanResponder: () => true,
        onPanResponderGrant: (evt: GestureResponderEvent, gestureState: PanResponderGestureState) => {
          setIsDragging(true);
        },
        onPanResponderMove: (evt: GestureResponderEvent, gestureState: PanResponderGestureState) => {
          let newPosition = currentPosition + gestureState.dx;
          newPosition = Math.max(0, Math.min(newPosition, maxPosition));
          setCurrentPosition(newPosition);
          sliderPosition.setValue(newPosition);
          if (onSliderMoved) {
            onSliderMoved(newPosition / maxPosition);
          }
        },
        onPanResponderRelease: (evt: GestureResponderEvent, gestureState: PanResponderGestureState) => {
          setIsDragging(false);
          if (onSliderCompleted) {
            onSliderCompleted(currentPosition / maxPosition);
          }
        },
      }),
    [currentPosition, maxPosition, onSliderCompleted, onSliderMoved, sliderPosition]
  );

  return (
    <View style={[styles.container, { width, height: height + 60 }]}>
      <View style={[styles.imageContainer, { width, height }]}>
        <Image
          source={{ uri: backgroundImageUrl }}
          style={styles.backgroundImage}
          resizeMode="cover"
        />
      </View>
      <View style={[styles.track, { width: width - 20, height: trackHeight }]}>
        <Animated.View
          {...panResponder.panHandlers}
          style={[
            styles.thumb,
            {
              width: thumbSize,
              height: thumbSize + 10,
              transform: [{ translateX: sliderPosition }],
            },
          ]}
        >
          <Image
            source={{ uri: sliderImageUrl }}
            style={styles.thumbImage}
            resizeMode="contain"
          />
        </Animated.View>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    backgroundColor: '#f5f5f5',
    borderRadius: 8,
    overflow: 'hidden',
  },
  imageContainer: {
    overflow: 'hidden',
  },
  backgroundImage: {
    width: '100%',
    height: '100%',
  },
  track: {
    marginTop: 15,
    marginHorizontal: 10,
    backgroundColor: '#e0e0e0',
    borderRadius: 2,
    position: 'relative',
  },
  thumb: {
    position: 'absolute',
    top: -5,
    left: 0,
    backgroundColor: '#fff',
    borderRadius: 4,
    elevation: 3,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.25,
    shadowRadius: 3.84,
    justifyContent: 'center',
    alignItems: 'center',
  },
  thumbImage: {
    width: 40,
    height: 40,
  },
});

export default SliderCaptcha;
