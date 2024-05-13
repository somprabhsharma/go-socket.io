package socketio

import (
	"math"
	"math/rand"
)

type RetryStrategy struct {
	ms       float64
	max      float64
	factor   float64
	jitter   float64
	attempts float64
}

func NewBackOff(opts RetryStrategy) *RetryStrategy {
	return &RetryStrategy{
		ms:       opts.ms,
		max:      opts.max,
		factor:   opts.factor,
		jitter:   opts.jitter,
		attempts: opts.attempts,
	}
}

func (b *RetryStrategy) Duration() float64 {
	ms := b.ms * math.Pow(b.factor, b.attempts)
	b.attempts++

	if b.jitter > 0 {
		randVal := rand.Float64()
		deviation := math.Floor(randVal * b.jitter * ms)
		jitterDecision := int(math.Floor(randVal*10)) & 1
		if jitterDecision == 0 {
			ms -= deviation
		} else {
			ms += deviation
		}
	}

	return math.Min(ms, b.max)
}

func (b *RetryStrategy) Reset() {
	b.attempts = 0
}

func (b *RetryStrategy) SetMin(ms float64) {
	b.ms = ms
}

func (b *RetryStrategy) SetMax(max float64) {
	b.max = max
}
func (b *RetryStrategy) SetJitter(jitter float64) {
	b.jitter = jitter
}
