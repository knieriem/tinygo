//go:build nucleog031k8

package machine

import (
	"device/stm32"
	"runtime/interrupt"
)

const (
	// Arduino Pins
	A0 = PA0  // ADC_IN0
	A1 = PA1  // ADC_IN1
	A2 = PA4  // ADC_IN4
	A3 = PA5  // ADC_IN5
	A4 = PA12 // ADC_IN16 || I2C2_SDA
	A5 = PA11 // ADC_IN15 || I2C2_SCL
	A6 = PA6  // ADC_IN6
	A7 = PA7  // ADC_IN7

	D0  = PB7 // USART1_RX
	D1  = PB6 // USART1_TX
	D2  = PA15
	D3  = PB1  // TIM3_CH4
	D4  = PA10 // TIM1_CH3 / I2C1_SDA
	D5  = PA9  // TIM1_CH2 / I2C1_SCL
	D6  = PB0  // TIM3_CH2
	D7  = PB2
	D8  = PB8
	D9  = PA8 // TIM1_CH1
	D10 = PB9 // SPI_CS || TIM17_CH1
	D11 = PB5 // SPI1_MOSI || TIM3_CH2
	D12 = PB4 // SPI1_MISO
	D13 = PB3 // SPI1_SCK
)

const (
	LED         = LED_BUILTIN
	LED_BUILTIN = LED_GREEN
	LED_GREEN   = PC6
)

const (
	BUTTON = PF2
)

const (
	// UART pins
	// PA2 and PA3 are connected to the ST-Link Virtual Com Port (VCP)
	UART_TX_PIN = PA2
	UART_RX_PIN = PA3

	// SPI
	SPI1_SCK_PIN = PB3
	SPI1_SDI_PIN = PB5
	SPI1_SDO_PIN = PB4
	SPI0_SCK_PIN = SPI1_SCK_PIN
	SPI0_SDI_PIN = SPI1_SDI_PIN
	SPI0_SDO_PIN = SPI1_SDO_PIN

	// I2C pins
	// PA11 and PA12 are mapped to CN4 pin 7 and CN4 pin 8 respectively
	I2C0_SCL_PIN  = PA11
	I2C0_SDA_PIN  = PA12
	I2C0_ALT_FUNC = 6
)

var (
	// USART2 (alt func 1) is the hardware serial port connected to the
	// onboard ST-LINK debugger to be exposed as virtual COM
	// port over USB on Nucleo boards. We use LPUART here, though,
	// since it is available on the same pins (PA2/PA3) and provides
	// RX/TX FIFOs. For g0 controllers < g07x the USART2 is available
	// in a BASIC implementation only, which lacks FIFOs.
	UART1  = &_UART1
	_UART1 = UART{
		Buffer:            NewRingBuffer(),
		Bus:               stm32.LPUART,
		TxAltFuncSelector: 6,
		RxAltFuncSelector: 6,
	}
	DefaultUART = UART1

	// I2C2 is documented, alias to I2C0 as well
	I2C2 = &I2C{
		Bus:             stm32.I2C2,
		AltFuncSelector: 6,
	}
	I2C0 = I2C2

	// SPI
	SPI0 = SPI{
		Bus:             stm32.SPI1,
		AltFuncSelector: 0,
	}
	SPI1 = &SPI0
)

func init() {
	UART1.Interrupt = interrupt.New(stm32.IRQ_USART3_USART4_LPUART1, _UART1.handleInterrupt)
}
