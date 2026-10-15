package timing

import (
	"time"
)

// CANBittiming defines the CAN bit timing values.
// The length of a time quantum, as mentioned below,
// is the oscillator period multiplied with the prescaler value
// (and, depending on the device, the value of an extra,
// device-internal divider).
type BitTiming struct {
	Prescaler uint16 // the CAN peripheral's programmable prescaler
	TSeg1     uint16 // length of the prop. time seg. + phase buffer seg. 1
	TSeg2     uint8  // length of the phase buffer segment 2, in tq
	SJW       uint8  // resynchronization jump width, in tq
}

// Constraints defines device specific limits and properties.
// CAN bit timing configuration registers of a specific device
// allow values for the time segments, the prescaler, and sjw
// within defined ranges only, dependent on the number of bits
// available for each value.
type Constraints struct {
	PropSegMax uint8
	TSeg1Min   uint8
	TSeg1Max   uint16
	TSeg2Min   uint8
	TSeg2Max   uint8

	SJWMax uint8

	PrescalerMin  uint16
	PrescalerMax  uint16
	PrescalerIncr uint16
}

// Controller contains the device specific limits for the
// bit timings of the nominal and data bitrates.
type Controller struct {
	// Clock is the Controller's default clock frequency
	Clock uint32

	Nominal Constraints
	Data    *Constraints
}

// NqMax returns the maxmimum number of time quanta that
// can be used on a specific device.
func (dev *Constraints) NqMax() int {
	return 1 + int(dev.TSeg1Max) + int(dev.TSeg2Max)
}

// The value of a time quantum. Note that if a device applies, in addition to the
// prescaler, an extra divider to fOsc, the argument to Tq must be divided by
// that value. For instance, a MCP2515 contains an internal division by two,
// so in case of a 16 MHz oscillator the argument fOsc should be 8 MHz.
func (t *BitTiming) CalcTq(fCAN uint32) time.Duration {
	return time.Second * time.Duration(t.Prescaler) / time.Duration(fCAN)
}

// Number of time quanta.
func (t *BitTiming) Nq() int {
	return 1 + int(t.TSeg1) + int(t.TSeg2)
}

// Bitrate calculates the resulting CAN bitrate for the given fOsc.
// See the comment for the .Tq method for how to reduce fOsc in case
// of an extra device internal divider.
func (t *BitTiming) Bitrate(fOsc uint32) uint32 {
	return fOsc / uint32(t.Prescaler) / uint32(t.Nq())
}

// SamplePoint returns the sample point defined by the BitTiming values.
func (t *BitTiming) SamplePoint() Frac1000 {
	return calcFrac(int(t.TSeg1), t.Nq())
}

// EnsureSJW first initializes the SJW field in case it is still zero.
// In this case it will set SJW to a ratio of the bit length,
// if ratio1000 is given, or to the value of TSeg2.
// It also verifies that SJW is between bounds 1, TSeg2 and maxSJW.
func (t *BitTiming) EnsureSJW(maxSJW uint8, f Frac1000) {
	sjw := t.SJW
	if sjw == 0 {
		if r := f; r != 0 {
			nq := t.Nq()
			sjw = uint8(f.apply(nq))
		} else {
			sjw = t.TSeg2
		}
	}
	sjw = max(1, sjw)
	if t.TSeg2 != 0 {
		sjw = min(sjw, t.TSeg2)
	}
	if maxSJW != 0 {
		sjw = min(sjw, maxSJW)
	}
	t.SJW = sjw
}
