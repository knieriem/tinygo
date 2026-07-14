//go:build stm32h757_cm7 || stm32h723

package machine

import "device/stm32"

const (
	// STM32H7 SRAMCAN base address
	sramcanBase uintptr = 0x4000AC00

	// FDCAN1..3
	numFDCANInstances = 3
)

func enableFDCANClock() {
	// clock FDCAN from PLL1
	stm32.RCC.SetD2CCIP1R_FDCANSEL(stm32.RCC_D2CCIP1R_FDCANSEL_PLL1_Q)

	// FDCAN clock is on APB1
	stm32.RCC.SetAPB1HENR_FDCANEN(1) // TODO-VERIFY exact field name in your SVD-generated bindings
}

func (can *CAN) setClockDiv() {
}

func (can *CAN) setCCCR_BRSE(value uint32) {
	// FIXME: should be renamed to BRSE in upstream svd (?)
	can.Bus.SetCCCR_BSE(value)
}

func (can *CAN) setILS_RF0NL(value uint32) {
	can.Bus.SetILS_RF0NL(value)
}

func (can *CAN) configFilterGlobal() {
	// GFC: reject/accept behavior for non-matching frames.
	// This should be made configurable.
	can.Bus.SetGFC_ANFE(0) // 0 = accept in Rx FIFO0 if no match
	can.Bus.SetGFC_ANFS(0)
}

func (can *CAN) ensureRAMConfig() error {
	if can.RAMConfig == nil {
		return errCANMissingRAMConfig
	}
	return nil
}

func (can *CAN) configMessageRAM() {
	rc := can.RAMConfig
	ro := &can.ramOffsets

	// program every RAM section's start address
	can.Bus.SIDFC.Set((uint32(rc.StdFilterLen) << 16) | ro.FLSSA.regVal())
	can.Bus.XIDFC.Set((uint32(rc.ExtFilterLen) << 16) | ro.FLESA.regVal())
	can.Bus.RXF0C.Set((uint32(rc.RxFIFO0Len) << 16) | ro.RF0SA.regVal())
	can.Bus.RXF1C.Set((uint32(rc.RxFIFO1Len) << 16) | ro.RF1SA.regVal())
	can.Bus.RXBC.Set(ro.RBSA.regVal())
	can.Bus.TXEFC.Set((uint32(rc.TxEventFIFOLen) << 16) | ro.EFSA.regVal())
	can.Bus.TXBC.Set((uint32(rc.TxFIFOQueueLen) << 24) | ro.TFQSA.regVal())

	// RXESC/TXESC: element data field size. 0x7 = 64 bytes (max FD payload)
	// for every FIFO/buffer kind used.
	can.Bus.RXESC.Set(0x7 | (0x7 << 4) | (0x7 << 8))
	can.Bus.TXESC.Set(0x7)
}
