package audio

import (
	"encoding/binary"
	"math"

	"github.com/fluentfox/api/internal/common"
)

const (
	wavHeaderBytes  = 12
	chunkHeaderSize = 8
	pcmFormat       = 1
	bitsPerSample16 = 16
	maxSample16     = math.MaxInt16
)

// normalizePeak scales 16-bit PCM WAV so its loudest sample sits at
// AUDIO_TARGET_PEAK_DBFS, with the gain capped at AUDIO_MAX_GAIN so near-silence
// is not blown up into noise. Clips in any other format, or already at the
// target, are returned unchanged. It edits a copy, never the input.
func normalizePeak(wav []byte) []byte {
	data, ok := pcm16Data(wav)
	if !ok || len(data) < 2 {
		return wav
	}
	peak := 0
	for i := 0; i+1 < len(data); i += 2 {
		v := int(int16(binary.LittleEndian.Uint16(data[i:])))
		if v < 0 {
			v = -v
		}
		peak = max(peak, v)
	}
	if peak == 0 {
		return wav
	}
	target := maxSample16 * math.Pow(10, common.AUDIO_TARGET_PEAK_DBFS/20)
	gain := math.Min(target/float64(peak), common.AUDIO_MAX_GAIN)
	if gain <= 1 {
		return wav
	}
	out := append([]byte(nil), wav...)
	offset := len(wav) - len(data)
	for i := 0; i+1 < len(data); i += 2 {
		v := float64(int16(binary.LittleEndian.Uint16(data[i:]))) * gain
		v = math.Max(math.MinInt16, math.Min(maxSample16, math.Round(v)))
		binary.LittleEndian.PutUint16(out[offset+i:], uint16(int16(v)))
	}
	return out
}

// pcm16Data returns the sample bytes of a 16-bit PCM WAV (a sub-slice of wav
// that runs to its end), walking the RIFF chunks to find "fmt " and "data".
func pcm16Data(wav []byte) ([]byte, bool) {
	if !isWAV(wav) {
		return nil, false
	}
	isPCM16 := false
	for pos := wavHeaderBytes; pos+chunkHeaderSize <= len(wav); {
		id := string(wav[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(wav[pos+4:]))
		body := pos + chunkHeaderSize
		switch {
		case id == "fmt " && body+16 <= len(wav):
			isPCM16 = binary.LittleEndian.Uint16(wav[body:]) == pcmFormat &&
				binary.LittleEndian.Uint16(wav[body+14:]) == bitsPerSample16
		case id == "data":
			end := min(body+size, len(wav))
			if !isPCM16 || end != len(wav) {
				return nil, false
			}
			return wav[body:end], true
		}
		pos = body + size + size%2
	}
	return nil, false
}
