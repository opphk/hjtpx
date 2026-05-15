package audio

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
)

const (
	SampleRate     = 16000
	BitsPerSample  = 16
	NumChannels    = 1
	DurationPerChar = 0.5
)

type VoiceType int

const (
	VoiceMale VoiceType = iota
	VoiceFemale
	VoiceElderly
)

const charset = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"

type Generator struct{}

func NewGenerator() *Generator {
	return &Generator{}
}

func (g *Generator) GenerateRandomCode(length int) string {
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func (g *Generator) GenerateWAVAudio(code string, voiceType VoiceType) ([]byte, int, error) {
	var audioBuffer bytes.Buffer

	numSamples := int(float64(len(code)) * DurationPerChar * float64(SampleRate))
	totalDuration := numSamples / SampleRate

	if err := g.writeWAVHeader(&audioBuffer, numSamples); err != nil {
		return nil, 0, fmt.Errorf("failed to write WAV header: %w", err)
	}

	for i, char := range code {
		samples := g.generateCharacterSamples(char, voiceType)

		startSample := int(float64(i) * DurationPerChar * float64(SampleRate))
		endSample := int(float64(i+1) * DurationPerChar * float64(SampleRate))
		if endSample > numSamples {
			endSample = numSamples
		}

		noiseStart := startSample + int(0.3*float64(SampleRate))
		noiseEnd := endSample - int(0.1*float64(SampleRate))

		for j := 0; j < numSamples; j++ {
			sample := int16(0)

			if j >= startSample && j < endSample {
				idx := j - startSample
				if idx < len(samples) {
					sample = samples[idx]
				}

				if j >= noiseStart && j < noiseEnd {
					noise := int16(rand.Intn(400) - 200)
					sample = g.mixSamples(sample, noise, 0.15)
				}

				envelope := g.getEnvelope(j-startSample, endSample-startSample)
				sample = g.multiplySample(sample, envelope)
			}

			if err := binary.Write(&audioBuffer, binary.LittleEndian, sample); err != nil {
				return nil, 0, fmt.Errorf("failed to write sample: %w", err)
			}
		}
	}

	return audioBuffer.Bytes(), totalDuration, nil
}

func (g *Generator) writeWAVHeader(w *bytes.Buffer, numSamples int) error {
	dataSize := numSamples * NumChannels * BitsPerSample / 8
	fileSize := 36 + dataSize

	header := struct {
		RIFF       [4]byte
		FileSize   uint32
		Wave       [4]byte
		Fmt        [4]byte
		FmtSize    uint32
		AudioFmt   uint16
		NumChannels uint16
		SampleRate uint32
		ByteRate   uint32
		BlockAlign uint16
		BitsPerSample uint16
		Data       [4]byte
		DataSize   uint32
	}{
		RIFF:         [4]byte{'R', 'I', 'F', 'F'},
		FileSize:     uint32(fileSize),
		Wave:         [4]byte{'W', 'A', 'V', 'E'},
		Fmt:          [4]byte{'f', 'm', 't', ' '},
		FmtSize:      16,
		AudioFmt:     1,
		NumChannels:  NumChannels,
		SampleRate:   SampleRate,
		ByteRate:     SampleRate * uint32(NumChannels) * uint32(BitsPerSample/8),
		BlockAlign:   NumChannels * BitsPerSample / 8,
		BitsPerSample: BitsPerSample,
		Data:         [4]byte{'d', 'a', 't', 'a'},
		DataSize:     uint32(dataSize),
	}

	if err := binary.Write(w, binary.LittleEndian, header); err != nil {
		return err
	}

	return nil
}

func (g *Generator) generateCharacterSamples(char rune, voiceType VoiceType) []int16 {
	samples := make([]int16, int(DurationPerChar*float64(SampleRate)))

	freq := g.getCharacterFrequency(char)
	baseAmplitude := int16(20000)

	switch voiceType {
	case VoiceMale:
		baseAmplitude = 22000
	case VoiceFemale:
		baseAmplitude = 18000
		freq *= 1.3
	case VoiceElderly:
		baseAmplitude = 16000
		freq *= 0.85
	}

	speedVariation := 0.9 + rand.Float64()*0.2
	freq *= speedVariation

	for i := 0; i < len(samples); i++ {
		t := float64(i) / float64(SampleRate)

		fundamental := math.Sin(2 * math.Pi * freq * t)

		harmonics := 0.0
		if voiceType == VoiceMale {
			harmonics = 0.3*math.Sin(2*math.Pi*freq*2*t) +
				0.15*math.Sin(2*math.Pi*freq*3*t) +
				0.08*math.Sin(2*math.Pi*freq*4*t)
		} else if voiceType == VoiceFemale {
			harmonics = 0.2*math.Sin(2*math.Pi*freq*2*t) +
				0.1*math.Sin(2*math.Pi*freq*3*t)
		}

		modulation := 1.0 + 0.1*math.Sin(2*math.Pi*5*t)
		wave := (fundamental + harmonics) * modulation

		tremolo := 1.0 + 0.05*math.Sin(2*math.Pi*6*t)

		sample := wave * float64(baseAmplitude) * tremolo

		samples[i] = int16(math.Max(math.Min(sample, 32767), -32768))
	}

	return samples
}

func (g *Generator) getCharacterFrequency(char rune) float64 {
	freqMap := map[rune]float64{
		'0': 480, '1': 520, '2': 560, '3': 600, '4': 640,
		'5': 680, '6': 720, '7': 760, '8': 800, '9': 840,
		'a': 880, 'b': 920, 'c': 960, 'd': 1000, 'e': 1040,
		'f': 1080, 'g': 1120, 'h': 1160, 'j': 1200, 'k': 1240,
		'm': 1280, 'n': 1320, 'p': 1360, 'q': 1400, 'r': 1440,
		's': 1480, 't': 1520, 'u': 1560, 'v': 1600, 'w': 1640,
		'x': 1680, 'y': 1720, 'z': 1760,
		'A': 1800, 'B': 1840, 'C': 1880, 'D': 1920, 'E': 1960,
		'F': 2000, 'G': 2040, 'H': 2080, 'J': 2120, 'K': 2160,
		'L': 2200, 'M': 2240, 'N': 2280, 'P': 2320, 'Q': 2360,
		'R': 2400, 'S': 2440, 'T': 2480, 'U': 2520, 'V': 2560,
		'W': 2600, 'X': 2640, 'Y': 2680, 'Z': 2720,
	}

	if freq, ok := freqMap[char]; ok {
		return freq
	}
	return 1000.0
}

func (g *Generator) getEnvelope(sampleIndex, totalSamples int) float64 {
	attackSamples := int(0.05 * float64(SampleRate))
	releaseSamples := int(0.1 * float64(SampleRate))

	if sampleIndex < attackSamples {
		return float64(sampleIndex) / float64(attackSamples)
	}

	releaseStart := totalSamples - releaseSamples
	if sampleIndex > releaseStart {
		return float64(totalSamples-sampleIndex) / float64(releaseSamples)
	}

	return 1.0
}

func (g *Generator) mixSamples(s1, s2 int16, ratio float64) int16 {
	mixed := float64(s1)*(1-ratio) + float64(s2)*ratio
	return int16(math.Max(math.Min(mixed, 32767), -32768))
}

func (g *Generator) multiplySample(s int16, multiplier float64) int16 {
	result := float64(s) * multiplier
	return int16(math.Max(math.Min(result, 32767), -32768))
}
