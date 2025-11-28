//go:build stm32h723 || stm32h757_cm7

package machine

import (
	"device/stm32"
	"runtime/volatile"
)

func getEXTIConfigRegister(pin uint8) *volatile.Register32 {
	switch (pin & 0xf) / 4 {
	case 0:
		return &stm32.SYSCFG.EXTICR1
	case 1:
		return &stm32.SYSCFG.EXTICR2
	case 2:
		return &stm32.SYSCFG.EXTICR3
	case 3:
		return &stm32.SYSCFG.EXTICR4
	}
	return nil
}

func enableEXTIConfigRegisters() {
	// Enable SYSCFG
	stm32.RCC.APB4ENR.SetBits(stm32.RCC_APB4ENR_SYSCFGEN)
}
