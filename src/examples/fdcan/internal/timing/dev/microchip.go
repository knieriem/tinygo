package dev

import "github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing"

// MCP2515Const defines the register value constraints of the MCP2515 CAN
// controller;
// see https://ww1.microchip.com/downloads/en/DeviceDoc/MCP2515-Stand-Alone-CAN-Controller-with-SPI-20001801J.pdf#page=43
var MCP2515 = &timing.Controller{
	Clock: 16e6 / 2,

	Nominal: timing.Constraints{
		PropSegMax:   8,
		TSeg1Max:     16,
		TSeg2Min:     2,
		TSeg2Max:     8,
		SJWMax:       4,
		PrescalerMax: 64,
	},
}

// MCP2518FD defines the register value constraints of the MCP2518FD CAN
// controller;
// see https://ww1.microchip.com/downloads/aemDocuments/documents/OTH/ProductDocuments/DataSheets/External-CAN-FD-Controller-with-SPI-Interface-DS20006027B.pdf#page=28
var MCP2518FD = &timing.Controller{
	Clock: 40e6,
	Nominal: timing.Constraints{
		TSeg1Max:     256,
		TSeg2Max:     128,
		SJWMax:       128,
		PrescalerMax: 256,
	},
	Data: &timing.Constraints{
		TSeg1Max:     32,
		TSeg2Max:     16,
		SJWMax:       16,
		PrescalerMax: 256,
	},
}
