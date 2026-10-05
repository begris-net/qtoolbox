# ape

`golift.io/ape` is a Go module that encodes and decodes Monkey's Audio (APE) files.

Cue-sheet splitting and copying existing frames exist in `golift.io/xtractr`.
This module owns the codec. A checked encode item is implemented. Tests
confirm encode by decoding the file with ffmpeg, and confirm decode by
comparing the PCM.

## Parity with Monkey's Audio 13.26

### Encode

- [x] Version 3990 descriptor, header, and seek table
- [x] File MD5
- [x] Fast (1000), 73728 samples per frame
- [x] 16-bit little-endian PCM
- [x] Mono
- [x] Stereo, including mid/side, silence, and pseudo-stereo
- [x] Multiple frames and a short final frame
- [x] Flag so a decoder can build a WAV header
- [x] Normal (2000), neural-network filter 16/11
- [x] High (3000), neural-network filter 64/11
- [x] Extra high (4000), filters 256/13 then 32/10, frame size ×4
- [x] Insane (5000), filters 1280/15 then 256/13 then 16/11, frame size ×16
- [x] 8-bit
- [x] 24-bit
- [x] 32-bit integer
- [x] 32-bit float (ffmpeg 9 cannot decode either 32-bit form)
- [x] 3 to 32 channels (ffmpeg only decodes mono and stereo)
- [x] Keep an original WAV, AIFF, W64, SND, or CAF header and footer
- [x] APE tags

```go
err := ape.Encode(file, pcm, ape.Stream{
    SampleRate: 44100,
    Channels:   2,
    Bits:       16,
}, nil)
```

`NewEncoder` writes the same file from PCM chunks. The sample count has to
be known first, because the seek table is written in front of the audio.

```go
enc, err := ape.NewEncoder(file, stream, samples, nil)
```

### Decode

- [x] Version 3990
- [x] File versions 3.95 through 3.97
- [ ] File versions 3.93 and 3.94 (_needs testing_)
- [x] APL image links (`DecodeFile` follows the link and returns that sample range)

```go
pcm, stream, err := ape.Decode(file)
```

`NewDecoder` reads the header and returns one PCM frame at a time. The
caller owns the reader.

```go
dec, err := ape.NewDecoder(file)
frame, err := dec.Next()
```

`pcm` is interleaved little-endian samples. A nil options value selects fast
compression and the usual 73728-sample frame.

`Decode` reads a version 3990 file, or a 3.93–3.99 file, and returns that
PCM. `DecodeFile` does the same for a path, and if the path is an APL link
it opens the image and returns only the linked sample range. Versions
before 3.93 are not decoded. 3.93 and 3.94 use the older predictor; that
path is in place and still needs a real file to check.

This module is BSD-3-Clause. See [LICENSE](LICENSE).
