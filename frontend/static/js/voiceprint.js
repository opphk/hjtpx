class VoiceprintCaptcha {
    constructor(options = {}) {
        this.options = {
            apiBase: options.apiBase || '/api/v1',
            onSuccess: options.onSuccess || (() => {}),
            onError: options.onError || (() => {}),
            onInit: options.onInit || (() => {}),
            complexity: options.complexity || 3,
            patternType: options.patternType || 'sequence'
        };
        
        this.session = null;
        this.audioContext = null;
        this.mediaRecorder = null;
        this.audioChunks = [];
        this.isRecording = false;
        this.recordedAudio = null;
    }

    async init() {
        await this.generateCaptcha();
        this.options.onInit(this.session);
        return this;
    }

    async generateCaptcha() {
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/voiceprint/create`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    complexity: this.options.complexity,
                    pattern_type: this.options.patternType
                })
            });

            const data = await response.json();
            
            if (data.code === 0 && data.data) {
                this.session = data.data;
                return this.session;
            } else {
                throw new Error(data.message || 'Failed to generate captcha');
            }
        } catch (error) {
            this.options.onError(error);
            throw error;
        }
    }

    async playPattern() {
        if (!this.session || !this.session.audio_data) {
            throw new Error('No captcha session available');
        }

        try {
            const audioData = atob(this.session.audio_data);
            const bytes = new Uint8Array(audioData.length);
            for (let i = 0; i < audioData.length; i++) {
                bytes[i] = audioData.charCodeAt(i);
            }
            
            const blob = new Blob([bytes], { type: 'audio/wav' });
            const url = URL.createObjectURL(blob);
            
            const audio = new Audio(url);
            await audio.play();
            
            audio.onended = () => {
                URL.revokeObjectURL(url);
            };

            return true;
        } catch (error) {
            this.options.onError(error);
            throw error;
        }
    }

    async startRecording() {
        try {
            const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
            this.mediaRecorder = new MediaRecorder(stream);
            this.audioChunks = [];

            this.mediaRecorder.ondataavailable = (event) => {
                this.audioChunks.push(event.data);
            };

            this.mediaRecorder.start();
            this.isRecording = true;
            return true;
        } catch (error) {
            this.options.onError(error);
            throw error;
        }
    }

    async stopRecording() {
        return new Promise((resolve, reject) => {
            if (!this.mediaRecorder || !this.isRecording) {
                reject(new Error('Not recording'));
                return;
            }

            this.mediaRecorder.onstop = () => {
                const blob = new Blob(this.audioChunks, { type: 'audio/wav' });
                this.recordedAudio = blob;
                this.isRecording = false;
                
                const tracks = this.mediaRecorder.stream.getTracks();
                tracks.forEach(track => track.stop());
                
                resolve(blob);
            };

            this.mediaRecorder.stop();
        });
    }

    async verify() {
        if (!this.session || !this.recordedAudio) {
            throw new Error('Recording required before verification');
        }

        try {
            const reader = new FileReader();
            const audioBase64 = await new Promise((resolve, reject) => {
                reader.onloadend = () => resolve(reader.result.split(',')[1]);
                reader.onerror = reject;
                reader.readAsDataURL(this.recordedAudio);
            });

            const features = await this.extractFeatures();
            
            const response = await fetch(`${this.options.apiBase}/captcha/voiceprint/verify`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    session_id: this.session.session_id,
                    voice_data: audioBase64,
                    features: features
                })
            });

            const data = await response.json();
            
            if (data.code === 0 && data.data) {
                const result = data.data;
                if (result.success) {
                    this.options.onSuccess(result);
                } else {
                    this.options.onError(new Error(result.message));
                }
                return result;
            } else {
                throw new Error(data.message || 'Verification failed');
            }
        } catch (error) {
            this.options.onError(error);
            throw error;
        }
    }

    async extractFeatures() {
        if (!this.recordedAudio) {
            return null;
        }

        try {
            const arrayBuffer = await this.recordedAudio.arrayBuffer();
            const audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const audioBuffer = await audioContext.decodeAudioData(arrayBuffer);
            
            const channelData = audioBuffer.getChannelData(0);
            
            const mfcc = this.calculateMFCC(channelData, audioBuffer.sampleRate);
            const fundamentalFreq = this.calculateFundamentalFreq(channelData, audioBuffer.sampleRate);
            const energy = this.calculateEnergy(channelData);

            return {
                mfcc: mfcc,
                fundamental_freq: fundamentalFreq,
                energy: energy,
                spectral_flux: [],
                formants: []
            };
        } catch (error) {
            console.error('Feature extraction failed:', error);
            return null;
        }
    }

    calculateMFCC(samples, sampleRate) {
        const frameSize = 512;
        const numFrames = Math.floor(samples.length / frameSize);
        const mfcc = new Array(13).fill(0);

        for (let i = 0; i < 13; i++) {
            const baseFreq = 100.0 + i * 50.0;
            let sum = 0;
            
            for (let j = 0; j < numFrames; j++) {
                let frameSum = 0;
                for (let k = 0; k < frameSize; k++) {
                    const idx = j * frameSize + k;
                    if (idx < samples.length) {
                        frameSum += Math.abs(samples[idx]);
                    }
                }
                sum += frameSum * Math.sin(2 * Math.PI * baseFreq * j / numFrames);
            }
            
            mfcc[i] = sum / numFrames;
        }

        return mfcc;
    }

    calculateFundamentalFreq(samples, sampleRate) {
        const minPeriod = Math.floor(sampleRate / 1000);
        const maxPeriod = Math.floor(sampleRate / 80);
        
        let maxCorrelation = 0;
        let fundamentalFreq = 0;

        for (let period = minPeriod; period < maxPeriod && period < samples.length / 2; period++) {
            let correlation = 0;
            let norm1 = 0;
            let norm2 = 0;

            for (let i = 0; i < samples.length - period && i < 1000; i++) {
                correlation += samples[i] * samples[i + period];
                norm1 += samples[i] * samples[i];
                norm2 += samples[i + period] * samples[i + period];
            }

            if (norm1 > 0 && norm2 > 0) {
                const normalizedCorr = correlation / (Math.sqrt(norm1) * Math.sqrt(norm2));
                if (normalizedCorr > maxCorrelation) {
                    maxCorrelation = normalizedCorr;
                    fundamentalFreq = sampleRate / period;
                }
            }
        }

        return fundamentalFreq;
    }

    calculateEnergy(samples) {
        let energy = 0;
        const count = Math.min(samples.length, 10000);
        
        for (let i = 0; i < count; i++) {
            energy += samples[i] * samples[i];
        }

        return Math.sqrt(energy / count);
    }

    async getOptions() {
        try {
            const response = await fetch(`${this.options.apiBase}/captcha/voiceprint/options`);
            const data = await response.json();
            return data.data || {};
        } catch (error) {
            console.error('Failed to get options:', error);
            return {};
        }
    }

    reset() {
        this.session = null;
        this.recordedAudio = null;
        this.audioChunks = [];
        this.isRecording = false;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = VoiceprintCaptcha;
}
