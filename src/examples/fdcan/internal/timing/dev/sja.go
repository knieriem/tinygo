package dev

import "github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing"

// SJA1000 defines register constraints for the SJA1000;
// see https://www.nxp.com/docs/en/data-sheet/SJA1000.pdf#page=50
var SJA1000 = &timing.Controller{
	Nominal: timing.Constraints{
		TSeg1Max:     16,
		TSeg2Max:     8,
		SJWMax:       4,
		PrescalerMax: 32,
	},
}
