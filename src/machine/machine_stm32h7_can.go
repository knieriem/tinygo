//go:build stm32h757_cm7 || stm32h723

package machine

import "device/stm32"

const (
	// STM32H7 SRAMCAN base address
	sramcanBase = 0x4000AC00

	// FDCAN1..3
	numFDCANInstances = 3
)

func enableFDCANClock() {
	// clock FDCAN from PLL1
	stm32.RCC.SetD2CCIP1R_FDCANSEL(stm32.RCC_D2CCIP1R_FDCANSEL_PLL1_Q)

	// FDCAN clock is on APB1
	stm32.RCC.SetAPB1HENR_FDCANEN(1) // TODO-VERIFY exact field name in your SVD-generated bindings
}

// instanceRAMOffset returns this instance's byte offset into the shared
// message RAM region.
func (can *CAN) instanceRAMOffset() uintptr {
	return uintptr(can.instance) * sramcanSize
}

func (can *CAN) sramBase() uintptr {
	return uintptr(sramcanBase) + can.instanceRAMOffset()
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

func (can *CAN) configMessageRAMLayout() {
	// explicitly program every RAM section's start address
	instBase := uint32(can.instanceRAMOffset())

	wordOff := instBase + uint32(sramcanFLSSA)
	can.Bus.SIDFC.Set((uint32(sramcanFLSNbr) << 16) | (wordOff << 2))

	wordOff = instBase + uint32(sramcanFLESA)
	can.Bus.XIDFC.Set((uint32(sramcanFLENbr) << 16) | (wordOff << 2))

	wordOff = instBase + uint32(sramcanRF0SA)
	can.Bus.RXF0C.Set((uint32(sramcanRF0Nbr) << 16) | (wordOff << 2))

	wordOff = instBase + uint32(sramcanRF1SA)
	can.Bus.RXF1C.Set((uint32(sramcanRF1Nbr) << 16) | (wordOff << 2))

	wordOff = instBase + uint32(sramcanTEFSA)
	can.Bus.TXEFC.Set((uint32(sramcanTEFNbr) << 16) | (wordOff << 2))

	wordOff = instBase + uint32(sramcanTFQSA)
	can.Bus.TXBC.Set((uint32(sramcanTFQNbr) << 24) | (wordOff << 2))

	// RXESC/TXESC: element data field size. 0x7 = 64 bytes (max FD payload)
	// for every FIFO/buffer kind used.
	can.Bus.RXESC.Set(0x7 | (0x7 << 4) | (0x7 << 8))
	can.Bus.TXESC.Set(0x7)
}
