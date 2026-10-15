package timing_test

import (
	"testing"
	"time"

	"github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing"
	"github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing/dev"
)

type test struct {
	dev     *timing.Controller
	constr  *timing.Constraints
	fCAN    uint32
	bitrate uint32
	sp      timing.Frac1000
	sjw     uint8

	// expected values
	bt *timing.BitTiming
	tq time.Duration
}

// Most of these tests stem from configurations used during previous years.
var tests = []test{
	{
		// example from: https: //ww1.microchip.com/downloads/en/DeviceDoc/MCP2515-Stand-Alone-CAN-Controller-with-SPI-20001801J.pdf#page=43
		dev:     dev.MCP2515,
		fCAN:    20e6 / 2,
		bitrate: 125e3,
		sp:      625,
		bt: &timing.BitTiming{
			Prescaler: 5,
			TSeg1:     9,
			TSeg2:     6,
			SJW:       4,
		},
	}, {
		dev:     dev.MCP2518FD,
		bitrate: 500e3,
		bt: &timing.BitTiming{
			Prescaler: 1,
			TSeg1:     69,
			TSeg2:     10,
			SJW:       10,
		},
		tq: 25 * time.Nanosecond,
	}, {
		dev:     dev.MCP2518FD,
		constr:  dev.MCP2518FD.Data,
		bitrate: 1e6,
		sp:      750,
		sjw:     1,
		bt: &timing.BitTiming{
			Prescaler: 1,
			TSeg1:     29,
			TSeg2:     10,
			SJW:       1,
		},
	},
}

func TestCalc(t *testing.T) {
	for i := range tests {
		test := &tests[i]
		dev := test.dev

		fCAN := dev.Clock
		if f := test.fCAN; f != 0 {
			fCAN = f
		}
		spec := test.constr
		if test.constr == nil {
			spec = &dev.Nominal
		}
		var bt timing.BitTiming
		err := timing.CalcBitTiming(&bt, fCAN, test.bitrate, test.sp, spec)
		if err != nil {
			t.Error(err)
			continue
		}
		bt.SJW = test.sjw
		bt.EnsureSJW(spec.SJWMax, 0)
		if test.bt != nil {
			if !bitTimingsEqual(test.bt, &bt) {
				t.Errorf("test %d: timing mismatch %#v != %#v", i, test.bt, bt)
			}
		}
		if xTq := test.tq; xTq != 0 {
			f := fCAN
			if tq := bt.CalcTq(f); tq != xTq {
				t.Errorf("test %d: tq mismatch: %v != %v", i, tq, xTq)
			}
		}
	}
}

func bitTimingsEqual(bt, bt2 *timing.BitTiming) bool {
	if bt.TSeg1 != bt2.TSeg1 {
		return false
	}
	if bt.TSeg2 != bt2.TSeg2 {
		return false
	}
	if bt.Prescaler != bt2.Prescaler {
		return false
	}
	if bt.SJW != bt2.SJW {
		return false
	}
	return true
}
