package vpn_proxy

import (
	"errors"
	"io"
	"math/rand"
	"strings"
	"time"

	"Hamburger/internal/config"
)

type obfsEngine struct {
	cfg  config.VpnObfsConfig
	mode string
	rng  *rand.Rand
}

const (
	obfsModeReflect = "reflect"
	obfsModeDrip    = "drip"
	obfsModeBurst   = "burst"
	obfsModePulse   = "pulse"
)

func newObfsEngine(cfg config.VpnObfsConfig) *obfsEngine {
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		mode = obfsModeReflect
	}
	return &obfsEngine{
		cfg:  cfg,
		mode: mode,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (o *obfsEngine) Copy(dst io.Writer, src io.Reader) (int64, error) {
	if o == nil || !o.cfg.Enabled {
		return io.Copy(dst, src)
	}
	switch o.mode {
	case obfsModeDrip:
		// drip 模式：小块持续滴流，延迟更均匀，整体速率更平滑
		return o.copyDrip(dst, src)
	case obfsModeBurst:
		// burst 模式：短时间密集发送，达到阈值后暂停
		return o.copyBurst(dst, src)
	case obfsModePulse:
		// pulse 模式：随机脉冲抖动，偶发更长延迟
		return o.copyPulse(dst, src)
	default:
		// reflect 模式：随机分片与延迟，保持基础流量形态
		return o.copyReflect(dst, src)
	}
}

func (o *obfsEngine) copyReflect(dst io.Writer, src io.Reader) (int64, error) {
	minChunk, maxChunk := o.getChunkRange(1024)
	minDelay, maxDelay := o.getDelayRange()
	buf := make([]byte, maxChunk)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			offset := 0
			for offset < n {
				chunk := o.randBetween(minChunk, maxChunk)
				if chunk > n-offset {
					chunk = n - offset
				}
				w, werr := dst.Write(buf[offset : offset+chunk])
				total += int64(w)
				if w <= 0 && werr == nil {
					return total, io.ErrUnexpectedEOF
				}
				offset += w
				if werr != nil {
					return total, werr
				}
				o.sleepRandom(minDelay, maxDelay)
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func (o *obfsEngine) copyDrip(dst io.Writer, src io.Reader) (int64, error) {
	minChunk, _ := o.getChunkRange(256)
	minDelay, maxDelay := o.getDelayRange()
	buf := make([]byte, minChunk)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			w, werr := dst.Write(buf[:n])
			total += int64(w)
			if w <= 0 && werr == nil {
				return total, io.ErrUnexpectedEOF
			}
			if werr != nil {
				return total, werr
			}
			o.sleepRandom(minDelay, maxDelay)
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func (o *obfsEngine) copyBurst(dst io.Writer, src io.Reader) (int64, error) {
	minChunk, maxChunk := o.getChunkRange(1024)
	minDelay, maxDelay := o.getDelayRange()
	burstBytes := maxChunk * 8
	if burstBytes < 8192 {
		burstBytes = 8192
	}
	buf := make([]byte, maxChunk)
	var total int64
	var burstCount int
	for {
		n, err := src.Read(buf)
		if n > 0 {
			offset := 0
			for offset < n {
				chunk := o.randBetween(minChunk, maxChunk)
				if chunk > n-offset {
					chunk = n - offset
				}
				w, werr := dst.Write(buf[offset : offset+chunk])
				total += int64(w)
				if w <= 0 && werr == nil {
					return total, io.ErrUnexpectedEOF
				}
				offset += w
				burstCount += w
				if werr != nil {
					return total, werr
				}
				if burstCount >= burstBytes {
					burstCount = 0
					o.sleepRandom(minDelay, maxDelay)
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func (o *obfsEngine) copyPulse(dst io.Writer, src io.Reader) (int64, error) {
	minChunk, maxChunk := o.getChunkRange(1024)
	minDelay, maxDelay := o.getDelayRange()
	buf := make([]byte, maxChunk)
	var total int64
	for {
		n, err := src.Read(buf)
		if n > 0 {
			offset := 0
			for offset < n {
				chunk := o.randBetween(minChunk, maxChunk)
				if chunk > n-offset {
					chunk = n - offset
				}
				w, werr := dst.Write(buf[offset : offset+chunk])
				total += int64(w)
				if w <= 0 && werr == nil {
					return total, io.ErrUnexpectedEOF
				}
				offset += w
				if werr != nil {
					return total, werr
				}
				delay := o.randBetween(minDelay, maxDelay)
				if delay > 0 && o.rng.Intn(5) == 0 {
					delay = delay * 2
				}
				if delay > 0 {
					time.Sleep(time.Duration(delay) * time.Millisecond)
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return total, nil
			}
			return total, err
		}
	}
}

func (o *obfsEngine) getChunkRange(defaultMin int) (int, int) {
	minChunk := o.cfg.MinChunkSize
	maxChunk := o.cfg.MaxChunkSize
	if minChunk <= 0 {
		minChunk = defaultMin
	}
	if maxChunk < minChunk {
		maxChunk = minChunk
	}
	return minChunk, maxChunk
}

func (o *obfsEngine) getDelayRange() (int, int) {
	minDelay := o.cfg.MinDelayMs
	maxDelay := o.cfg.MaxDelayMs
	if maxDelay < minDelay {
		maxDelay = minDelay
	}
	return minDelay, maxDelay
}

func (o *obfsEngine) randBetween(minValue int, maxValue int) int {
	if maxValue <= minValue {
		return minValue
	}
	return o.rng.Intn(maxValue-minValue+1) + minValue
}

func (o *obfsEngine) sleepRandom(minDelay int, maxDelay int) {
	delay := o.randBetween(minDelay, maxDelay)
	if delay > 0 {
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}
}
