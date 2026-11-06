package audio

import (
	"encoding/binary"
	"math"
	"testing"
)

// pcmWAV builds a minimal 16-bit mono WAV holding the given samples.
func pcmWAV(samples ...int16) []byte {
	data := make([]byte, 2*len(samples))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(data[2*i:], uint16(s))
	}
	b := []byte("RIFF\x00\x00\x00\x00WAVE")
	fmtChunk := make([]byte, 24)
	copy(fmtChunk, "fmt ")
	binary.LittleEndian.PutUint32(fmtChunk[4:], 16)
	binary.LittleEndian.PutUint16(fmtChunk[8:], pcmFormat)
	binary.LittleEndian.PutUint16(fmtChunk[10:], 1)
	binary.LittleEndian.PutUint32(fmtChunk[12:], 24000)
	binary.LittleEndian.PutUint32(fmtChunk[16:], 48000)
	binary.LittleEndian.PutUint16(fmtChunk[20:], 2)
	binary.LittleEndian.PutUint16(fmtChunk[22:], bitsPerSample16)
	b = append(b, fmtChunk...)
	b = append(b, []byte("data")...)
	b = binary.LittleEndian.AppendUint32(b, uint32(len(data)))
	return append(b, data...)
}

func samplesOf(t *testing.T, wav []byte) []int16 {
	t.Helper()
	data, ok := pcm16Data(wav)
	if !ok {
		t.Fatal("not 16-bit PCM WAV")
	}
	out := make([]int16, len(data)/2)
	for i := range out {
		out[i] = int16(binary.LittleEndian.Uint16(data[2*i:]))
	}
	return out
}

func TestNormalizePeak(t *testing.T) {
	target := int16(math.Round(maxSample16 * math.Pow(10, -3.0/20)))
	tests := []struct {
		name string
		in   []int16
		want []int16
	}{
		{"quiet clip is raised to the target peak", []int16{0, 2900, -5800}, []int16{0, target / 2, -target}},
		{"gain is capped for near-silence", []int16{10, -20}, []int16{80, -160}},
		{"loud clip is left alone", []int16{0, 32000}, []int16{0, 32000}},
		{"silence is left alone", []int16{0, 0}, []int16{0, 0}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := pcmWAV(tc.in...)
			got := samplesOf(t, normalizePeak(in))
			for i := range got {
				if diff := int(got[i]) - int(tc.want[i]); diff < -2 || diff > 2 {
					t.Errorf("samples = %v, want %v", got, tc.want)
					break
				}
			}
			if samplesOf(t, in)[len(tc.in)-1] != tc.in[len(tc.in)-1] {
				t.Error("input was modified")
			}
		})
	}
}

func TestNormalizePeakLeavesOtherFormatsAlone(t *testing.T) {
	notWAV := []byte("ID3 an mp3 file")
	if got := normalizePeak(notWAV); string(got) != string(notWAV) {
		t.Errorf("non-WAV changed: %q", got)
	}
}

func TestSpeedFor(t *testing.T) {
	for text, want := range map[string]float64{"あ": 0.75, "きゃ": 0.75, "あさ": 0.75, "いぬです。": 0.95} {
		if got := speedFor(text); got != want {
			t.Errorf("speedFor(%q) = %v, want %v", text, got, want)
		}
	}
}
