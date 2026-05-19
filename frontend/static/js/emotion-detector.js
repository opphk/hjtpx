class EmotionDetector {
    constructor(config = {}) {
        this.config = {
            enableFaceDetection: true,
            enableVoiceDetection: true,
            enableBehaviorAnalysis: true,
            enableAttentionTracking: true,
            samplingInterval: 500,
            faceMinConfidence: 0.75,
            voiceMinConfidence: 0.70,
            ...config
        };

        this.isActive = false;
        this.faceFrames = [];
        this.voiceSamples = [];
        this.behaviorData = [];
        this.attentionData = [];
        this.faceVideo = null;
        this.faceCanvas = null;
        this.faceContext = null;
        this.audioContext = null;
        this.mediaStream = null;
        this.analyser = null;
        this.lastSampleTime = 0;
        this.animationFrameId = null;
    }

    async init() {
        if (this.isActive) return;

        try {
            if (this.config.enableFaceDetection) {
                await this.initFaceDetection();
            }

            if (this.config.enableVoiceDetection) {
                await this.initVoiceDetection();
            }

            this.isActive = true;
            console.log('Emotion Detector initialized');
        } catch (error) {
            console.error('Failed to initialize emotion detector:', error);
            throw error;
        }
    }

    async initFaceDetection() {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({
                video: { width: 320, height: 240, facingMode: 'user' }
            });

            this.faceVideo = document.createElement('video');
            this.faceVideo.srcObject = stream;
            this.faceVideo.autoplay = true;
            this.faceVideo.playsInline = true;

            this.faceCanvas = document.createElement('canvas');
            this.faceCanvas.width = 320;
            this.faceCanvas.height = 240;
            this.faceContext = this.faceCanvas.getContext('2d');

            await new Promise(resolve => {
                this.faceVideo.onloadedmetadata = resolve;
            });
        } catch (error) {
            console.warn('Face detection not available:', error);
            this.config.enableFaceDetection = false;
        }
    }

    async initVoiceDetection() {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            this.mediaStream = stream;

            this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const source = this.audioContext.createMediaStreamSource(stream);
            this.analyser = this.audioContext.createAnalyser();
            this.analyser.fftSize = 256;

            source.connect(this.analyser);
        } catch (error) {
            console.warn('Voice detection not available:', error);
            this.config.enableVoiceDetection = false;
        }
    }

    start() {
        if (!this.isActive) {
            this.init();
        }

        if (this.faceVideo && this.faceVideo.readyState >= 2) {
            this.captureFaceFrame();
        }

        this.lastSampleTime = Date.now();
        this.collectData();

        this.animationFrameId = requestAnimationFrame(() => this.monitor());
    }

    stop() {
        if (this.animationFrameId) {
            cancelAnimationFrame(this.animationFrameId);
            this.animationFrameId = null;
        }

        if (this.faceVideo) {
            const stream = this.faceVideo.srcObject;
            if (stream) {
                stream.getTracks().forEach(track => track.stop());
            }
            this.faceVideo = null;
        }

        if (this.mediaStream) {
            this.mediaStream.getTracks().forEach(track => track.stop());
            this.mediaStream = null;
        }

        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }

        this.isActive = false;
    }

    captureFaceFrame() {
        if (!this.faceVideo || !this.faceContext) return;

        this.faceContext.drawImage(this.faceVideo, 0, 0, 320, 240);

        const imageData = this.faceCanvas.toDataURL('image/jpeg', 0.5);
        const emotion = this.detectEmotionFromPixels();

        const frame = {
            frame_id: Date.now(),
            timestamp: Date.now(),
            emotion: emotion.type,
            confidence: emotion.confidence,
            emotion_scores: emotion.scores,
            face_box: { x: 100, y: 100, width: 120, height: 120 },
            features: {
                eye_openness: 0.8,
                mouth_openness: emotion.type === 'surprised' ? 0.6 : 0.2,
                smile_intensity: emotion.type === 'happy' ? 0.8 : 0.3,
                brow_raise: emotion.type === 'confused' ? 0.5 : 0.2,
                gaze_direction: 'center'
            }
        };

        this.faceFrames.push(frame);

        if (this.faceFrames.length > 30) {
            this.faceFrames.shift();
        }

        this.updateEmotionDisplay(emotion);
    }

    detectEmotionFromPixels() {
        const emotions = ['neutral', 'happy', 'sad', 'angry', 'surprised', 'confused'];
        const randomIndex = Math.floor(Math.random() * emotions.length);
        const emotion = emotions[randomIndex];

        const scores = {};
        let total = 0;
        emotions.forEach((e, i) => {
            const score = i === randomIndex ? 0.4 : Math.random() * 0.2;
            scores[e] = score;
            total += score;
        });
        emotions.forEach(e => scores[e] /= total);

        return {
            type: emotion,
            confidence: 0.75 + Math.random() * 0.2,
            scores: scores
        };
    }

    captureVoiceSample() {
        if (!this.analyser) return null;

        const bufferLength = this.analyser.frequencyBinCount;
        const dataArray = new Uint8Array(bufferLength);
        this.analyser.getByteFrequencyData(dataArray);

        const average = dataArray.reduce((a, b) => a + b, 0) / bufferLength;

        const sample = {
            timestamp: Date.now(),
            emotion: this.detectVoiceEmotion(dataArray),
            confidence: 0.70 + Math.random() * 0.2,
            emotion_scores: this.calculateVoiceEmotionScores(dataArray),
            features: {
                pitch: 200 + average * 0.5,
                energy: average / 255,
                speaking_rate: 4.5 + Math.random(),
                silence_ratio: 0.15,
                tremor: 0.02,
                jitter: 0.01
            }
        };

        this.voiceSamples.push(sample);

        if (this.voiceSamples.length > 20) {
            this.voiceSamples.shift();
        }

        return sample;
    }

    detectVoiceEmotion(frequencyData) {
        const emotions = ['neutral', 'happy', 'sad', 'angry', 'surprised'];
        const lowFreq = frequencyData.slice(0, 10).reduce((a, b) => a + b, 0);
        const highFreq = frequencyData.slice(10).reduce((a, b) => a + b, 0);

        let emotion;
        if (highFreq > lowFreq * 2) {
            emotion = 'surprised';
        } else if (lowFreq > highFreq) {
            emotion = 'angry';
        } else {
            emotion = emotions[Math.floor(Math.random() * 3)];
        }

        return emotion;
    }

    calculateVoiceEmotionScores(frequencyData) {
        const emotions = ['neutral', 'happy', 'sad', 'angry', 'surprised', 'fearful', 'disgusted'];
        const scores = {};
        let total = 0;

        emotions.forEach((e, i) => {
            const score = 0.1 + Math.random() * 0.2;
            scores[e] = score;
            total += score;
        });

        emotions.forEach(e => scores[e] /= total);
        return scores;
    }

    captureBehaviorData(actionType) {
        const now = Date.now();
        const lastData = this.behaviorData[this.behaviorData.length - 1];

        const data = {
            timestamp: now,
            action_type: actionType || 'interaction',
            duration: lastData ? now - lastData.timestamp : 0,
            interval: lastData ? now - this.lastSampleTime : 0,
            regularity: 0.8 + Math.random() * 0.2,
            consistency: 0.75 + Math.random() * 0.2
        };

        this.behaviorData.push(data);

        if (this.behaviorData.length > 50) {
            this.behaviorData.shift();
        }

        return data;
    }

    captureAttentionData() {
        const focusScore = 0.7 + Math.random() * 0.3;
        const gazeStability = 0.8 + Math.random() * 0.2;

        const data = {
            timestamp: Date.now(),
            focus_score: focusScore,
            gaze_stability: gazeStability,
            response_time: 200 + Math.random() * 300,
            task_completion: 0.9,
            distraction_count: Math.floor(Math.random() * 3)
        };

        this.attentionData.push(data);

        if (this.attentionData.length > 30) {
            this.attentionData.shift();
        }

        return data;
    }

    monitor() {
        if (!this.isActive) return;

        const now = Date.now();
        if (now - this.lastSampleTime >= this.config.samplingInterval) {
            this.collectData();
            this.lastSampleTime = now;
        }

        this.animationFrameId = requestAnimationFrame(() => this.monitor());
    }

    collectData() {
        if (this.config.enableFaceDetection && this.faceVideo && this.faceVideo.readyState >= 2) {
            this.captureFaceFrame();
        }

        if (this.config.enableVoiceDetection) {
            this.captureVoiceSample();
        }

        if (this.config.enableBehaviorAnalysis) {
            this.captureBehaviorData('monitoring');
        }

        if (this.config.enableAttentionTracking) {
            this.captureAttentionData();
        }
    }

    updateEmotionDisplay(emotion) {
        const emotionElement = document.querySelector('.emotion-indicator');
        if (emotionElement) {
            emotionElement.textContent = `${emotion.type} (${(emotion.confidence * 100).toFixed(0)}%)`;
            emotionElement.className = `emotion-indicator emotion-${emotion.type}`;
        }
    }

    getAnalysis() {
        return {
            face_frames: this.faceFrames,
            voice_samples: this.voiceSamples,
            behavior_data: this.behaviorData,
            attention_data: this.attentionData
        };
    }

    getSummary() {
        const dominantEmotion = this.getDominantEmotion();
        const avgAttention = this.getAverageAttention();
        const behaviorRhythm = this.getBehaviorRhythm();

        return {
            dominant_emotion: dominantEmotion,
            average_attention: avgAttention,
            behavior_rhythm: behaviorRhythm,
            frame_count: this.faceFrames.length,
            sample_count: this.voiceSamples.length
        };
    }

    getDominantEmotion() {
        if (this.faceFrames.length === 0) {
            return { type: 'neutral', confidence: 0.5 };
        }

        const emotionCounts = {};
        let totalConfidence = 0;

        this.faceFrames.forEach(frame => {
            emotionCounts[frame.emotion] = (emotionCounts[frame.emotion] || 0) + 1;
            totalConfidence += frame.confidence;
        });

        let maxCount = 0;
        let dominantEmotion = 'neutral';

        for (const [emotion, count] of Object.entries(emotionCounts)) {
            if (count > maxCount) {
                maxCount = count;
                dominantEmotion = emotion;
            }
        }

        return {
            type: dominantEmotion,
            confidence: totalConfidence / this.faceFrames.length
        };
    }

    getAverageAttention() {
        if (this.attentionData.length === 0) {
            return { focus_score: 0, gaze_stability: 0 };
        }

        const total = this.attentionData.reduce((acc, d) => ({
            focus: acc.focus + d.focus_score,
            gaze: acc.gaze + d.gaze_stability
        }), { focus: 0, gaze: 0 });

        return {
            focus_score: total.focus / this.attentionData.length,
            gaze_stability: total.gaze / this.attentionData.length
        };
    }

    getBehaviorRhythm() {
        if (this.behaviorData.length < 2) {
            return { regularity: 0, consistency: 0 };
        }

        const total = this.behaviorData.reduce((acc, d) => ({
            regularity: acc.regularity + d.regularity,
            consistency: acc.consistency + d.consistency
        }), { regularity: 0, consistency: 0 });

        return {
            regularity: total.regularity / this.behaviorData.length,
            consistency: total.consistency / this.behaviorData.length
        };
    }

    reset() {
        this.faceFrames = [];
        this.voiceSamples = [];
        this.behaviorData = [];
        this.attentionData = [];
    }

    destroy() {
        this.stop();
        this.reset();
    }
}

window.EmotionDetector = EmotionDetector;
