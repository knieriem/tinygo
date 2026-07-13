//go:build stm32g0b1

package machine

import "device/stm32"

const (
	// STM32G0B1 SRAMCAN base address
	sramcanBase = 0x4000B400

	numFDCANInstances = 2
)

// enableFDCANClock enables the FDCAN peripheral clock
func enableFDCANClock() {
	// FDCAN clock is on APB1
	stm32.RCC.SetAPBENR1_FDCANEN(1)
}

func (can *CAN) sramBase() uintptr {
	if can.Bus == stm32.FDCAN2 {
		return uintptr(sramcanBase) + sramcanSize
	}
	return uintptr(sramcanBase)
}

func (can *CAN) setClockDiv() {
	if can.Bus == stm32.FDCAN1 {
		can.Bus.SetCKDIV_PDIV(0) // No clock division.
	}
}

func (can *CAN) setCCCR_BRSE(value uint32) {
	// FIXME: should be renamed to BRSE in upstream svd (?)
	can.Bus.SetCCCR_BRSE(value)
}

func (can *CAN) setILS_RF0NL(value uint32) {
	can.Bus.SetILS_RxFIFO0(0)
}

func (can *CAN) configFilterGlobal() {
	// Set filter list sizes: LSS[20:16], LSE[27:24].
	// Note: this implicitely sets both ANFS and ANFE to 0b00.
	rxgfc := can.Bus.RXGFC.Get()
	rxgfc &= ^uint32(0x0F1F0000)
	rxgfc |= uint32(sramcanFLSNbr) << 16
	rxgfc |= uint32(sramcanFLENbr) << 24
	can.Bus.RXGFC.Set(rxgfc)
}

func (can *CAN) configMessageRAMLayout() {
	// Nothing to do here. The G0x1 series uses a simplified version of the
	// M_CAN silicon, the Message RAM layout is fixed.
}
