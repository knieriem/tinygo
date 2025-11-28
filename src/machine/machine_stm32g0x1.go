//go:build stm32g0x1

package machine

// Peripheral abstraction layer for the stm32g0

import (
	"device/stm32"
	"runtime/interrupt"
	"runtime/volatile"
	"unsafe"
)

const (
	AF0_SYSTEM_SPI1_USART2_LPTIM_TIM21 = 0
	AF1_SPI1_I2C1_LPTIM                = 1
	AF2_LPTIM_TIM2                     = 2
	AF3_I2C1                           = 3
	AF4_I2C1_USART2_LPUART1_TIM22      = 4
	AF5_TIM2_21_22                     = 5
	AF6_LPUART1                        = 6
	AF7_COMP1_2                        = 7
)

// Enable peripheral clock
func enableAltFuncClock(bus unsafe.Pointer) {
	switch bus {
	case unsafe.Pointer(stm32.PWR): // Power interface clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_PWREN)
	case unsafe.Pointer(stm32.I2C2): // I2C2 clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_I2C2EN)
	case unsafe.Pointer(stm32.I2C1): // I2C1 clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_I2C1EN)
	case unsafe.Pointer(stm32.USART2): // USART2 clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_USART2EN)
	case unsafe.Pointer(stm32.SPI2): // SPI2 clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_SPI2EN)
	case unsafe.Pointer(stm32.LPUART): // LPUART clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_LPUART1EN)
	case unsafe.Pointer(stm32.WWDG): // Window watchdog clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_WWDGEN)
	case unsafe.Pointer(stm32.TIM17): // TIM17 clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_TIM17EN)
	case unsafe.Pointer(stm32.TIM16): // TIM16 clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_TIM16EN)
	case unsafe.Pointer(stm32.TIM14): // TIM14 clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_TIM14EN)
	case unsafe.Pointer(stm32.TIM3): // TIM3 clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_TIM3EN)
	case unsafe.Pointer(stm32.TIM2): // TIM2 clock enable
		stm32.RCC.APBENR1.SetBits(stm32.RCC_APBENR1_TIM2EN)
	case unsafe.Pointer(stm32.SYSCFG): // System configuration controller clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_SYSCFGEN)
	case unsafe.Pointer(stm32.SPI1): // SPI1 clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_SPI1EN)
	case unsafe.Pointer(stm32.ADC): // ADC clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_ADCEN)
	case unsafe.Pointer(stm32.USART1): // USART1 clock enable
		stm32.RCC.APBENR2.SetBits(stm32.RCC_APBENR2_USART1EN)
	}
}

// ---------- Timer related code
// FIXME: Alternate functions for timer channels have not been adjusted yet.
var (
	TIM2 = TIM{
		EnableRegister: &stm32.RCC.APBENR1,
		EnableFlag:     stm32.RCC_APBENR1_TIM2EN,
		Device:         stm32.TIM2,
		Channels: [4]TimerChannel{
			TimerChannel{Pins: []PinFunction{{PA0, AF2_LPTIM_TIM2}, {PA5, AF5_TIM2_21_22}, {PA8, AF5_TIM2_21_22}, {PA15, AF5_TIM2_21_22}}},
			TimerChannel{Pins: []PinFunction{{PA1, AF2_LPTIM_TIM2}, {PB3, AF2_LPTIM_TIM2}}},
			TimerChannel{Pins: []PinFunction{{PA2, AF2_LPTIM_TIM2}, {PB0, AF5_TIM2_21_22}, {PB10, AF2_LPTIM_TIM2}}},
			TimerChannel{Pins: []PinFunction{{PA3, AF2_LPTIM_TIM2}, {PB1, AF5_TIM2_21_22}, {PB11, AF2_LPTIM_TIM2}}},
		},
		busFreq: APB1_TIM_FREQ,
	}

	TIM3 = TIM{
		EnableRegister: &stm32.RCC.APBENR1,
		EnableFlag:     stm32.RCC_APBENR1_TIM3EN,
		Device:         stm32.TIM3,
		Channels: [4]TimerChannel{
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
		},
		busFreq: APB1_TIM_FREQ,
	}

	TIM14 = TIM{
		EnableRegister: &stm32.RCC.APBENR2,
		EnableFlag:     stm32.RCC_APBENR2_TIM14EN,
		Device:         stm32.TIM14,
		Channels: [4]TimerChannel{
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
		},
		busFreq: APB2_TIM_FREQ,
	}

	TIM16 = TIM{
		EnableRegister: &stm32.RCC.APBENR2,
		EnableFlag:     stm32.RCC_APBENR2_TIM16EN,
		Device:         stm32.TIM16,
		Channels: [4]TimerChannel{
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
		},
		busFreq: APB2_TIM_FREQ,
	}
	TIM17 = TIM{
		EnableRegister: &stm32.RCC.APBENR2,
		EnableFlag:     stm32.RCC_APBENR2_TIM17EN,
		Device:         stm32.TIM17,
		Channels: [4]TimerChannel{
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
			TimerChannel{Pins: []PinFunction{}},
		},
		busFreq: APB2_TIM_FREQ,
	}
)

func (t *TIM) registerUPInterrupt() interrupt.Interrupt {
	switch t {
	case &TIM2:
		return interrupt.New(stm32.IRQ_TIM2, TIM2.handleUPInterrupt)
	case &TIM3:
		return interrupt.New(stm32.IRQ_TIM3, TIM3.handleUPInterrupt)
	case &TIM14:
		return interrupt.New(stm32.IRQ_TIM14, TIM14.handleUPInterrupt)
	case &TIM16:
		return interrupt.New(stm32.IRQ_TIM16, TIM16.handleUPInterrupt)
	case &TIM17:
		return interrupt.New(stm32.IRQ_TIM17, TIM17.handleUPInterrupt)
	}

	return interrupt.Interrupt{}
}

func (t *TIM) registerOCInterrupt() interrupt.Interrupt {
	switch t {
	case &TIM2:
		return interrupt.New(stm32.IRQ_TIM2, TIM2.handleOCInterrupt)
	case &TIM3:
		return interrupt.New(stm32.IRQ_TIM3, TIM3.handleOCInterrupt)
	case &TIM14:
		return interrupt.New(stm32.IRQ_TIM14, TIM14.handleOCInterrupt)
	case &TIM16:
		return interrupt.New(stm32.IRQ_TIM16, TIM16.handleOCInterrupt)
	case &TIM17:
		return interrupt.New(stm32.IRQ_TIM17, TIM17.handleOCInterrupt)
	}

	return interrupt.Interrupt{}
}

func (t *TIM) enableMainOutput() {
	// nothing to do - no BDTR register
}

type arrtype = uint32
type arrRegType = volatile.Register32

const (
	ARR_MAX = 0x10000
	PSC_MAX = 0x10000
)

func handlePinInterrupt(pin uint8) {
	// The pin abstraction doesn't differentiate pull-up
	// events from pull-down events, so combine them to
	// a single call here.

	if stm32.EXTI.RPR1.HasBits(1<<pin) || stm32.EXTI.FPR1.HasBits(1<<pin) {
		// Writing 1 to the pending register clears the
		// pending flag for that bit
		stm32.EXTI.RPR1.Set(1 << pin)
		stm32.EXTI.FPR1.Set(1 << pin)

		callback := pinCallbacks[pin]
		if callback != nil {
			callback(interruptPins[pin])
		}
	}
}
