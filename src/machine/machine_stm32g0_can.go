//go:build stm32g0b1

package machine

import "device/stm32"

const (
	// STM32G0B1 SRAMCAN base address
	sramcanBase = 0x4000B400
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
