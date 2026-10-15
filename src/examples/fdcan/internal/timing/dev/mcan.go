package dev

import "github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing"

// MCAN defines the register value constraints of an [M_CAN] peripheral.
//
// [M_CAN]: https://www.bosch-semiconductors.com/products/ip-modules/can-ip-modules/m-can/
var MCAN = &timing.Controller{
	Clock: 40e6,
	Nominal: timing.Constraints{
		PrescalerMax: 512,
		TSeg1Max:     256,
		TSeg2Max:     128,
		SJWMax:       128,
	},
	Data: &timing.Constraints{
		PrescalerMax: 32,
		TSeg1Max:     32,
		TSeg2Max:     16,
		SJWMax:       16,
	},
}
