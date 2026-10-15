package timing

import (
	"errors"
)

const (
	tq      = 1
	syncSeg = 1 * tq
)

// CalcBitTiming calculates a bit timing with the desired location of the
// sample point for the given oscillator frequency and bitrate.
// Zero values may be used for the sample point,
// in which case sp=87.5 % are substituted.
func CalcBitTiming(t *BitTiming, fCAN, bitrate uint32, sp Frac1000, dev *Constraints) error {
	var minErrLoc = 1 << 16
	var bestTiming BitTiming

	incr := atLeast1(dev.PrescalerIncr)

	nqMax := dev.NqMax()
	nq0 := fCAN / bitrate
	if sp == 0 {
		sp = 875
	}

	for preSc := atLeast1(dev.PrescalerMin); preSc < dev.PrescalerMax; preSc += incr {
		nq := int(nq0 / uint32(preSc))
		if nq0%uint32(preSc) != 0 {
			continue
		}
		if nq <= 0 {
			break
		}
		if nq > nqMax {
			continue
		}
		ps2 := dev.constrainPhSeg2(sp.applyCompl(nq))
	again:
		// compute a tseg1
		tseg1 := nq - ps2 - syncSeg
		if tseg1 > int(dev.TSeg1Max) {
			if ps2 >= int(dev.TSeg2Max) {
				continue
			}
			ps2++
			goto again
		}
		// at least 1tq for each of propSeg and phSeg1 needed
		if tseg1 < 2 {
			break
		}

		errLoc := abs(int(calcFrac(1+tseg1, nq) - sp))
		if errLoc >= minErrLoc {
			continue
		}
		minErrLoc = errLoc

		bestTiming = BitTiming{
			Prescaler: uint16(preSc),
			TSeg1:     uint16(tseg1),
			TSeg2:     uint8(ps2),
		}
		if errLoc == 0 {
			break
		}
	}
	if bestTiming.TSeg2 == 0 {
		return ErrNoValidBitTimingFound
	}

	*t = bestTiming
	return nil
}

var ErrNoValidBitTimingFound = errors.New("can: no valid bit timing found")

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func atLeast1[T uint16 | int](v T) T {
	return max(1, v)
}

func (dev *Constraints) SplitTSeg1(tseg1, ps1start int) (prop, ps1 int) {
	ps1 = ps1start

	// propSeg gets the remaining tqs
	prop = tseg1 - ps1

	// adjust prop, if it is negative or zero
	for prop <= 0 {
		ps1--
		prop++
	}

	// adjust prop, if it is too large to be programmable
	maxProp := uint16(dev.PropSegMax)
	if maxProp == 0 {
		maxProp = dev.TSeg1Max / 2
	}
	for prop > int(maxProp) {
		prop--
		ps1++
	}
	return
}

func (dev *Constraints) constrainPhSeg2(ps2 int) int {
	minPhSeg2 := atLeast1(int(dev.TSeg2Min))

	if ps2 < minPhSeg2 {
		ps2 = minPhSeg2
	}
	if ps2 > int(dev.TSeg2Max) {
		ps2 = int(dev.TSeg2Max)
	}
	return ps2
}
