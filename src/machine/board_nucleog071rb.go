//go:build nucleog071rb

package machine

import (
	"device/stm32"
	"runtime/interrupt"
)

const (
	// Arduino Pins
	A0 = PA0 // ADC_IN0
	A1 = PA1 // ADC_IN1
	A2 = PA4 // ADC_IN4
	A3 = PB1 // ADC_IN9
	A4 = PB9 // ADC_IN15 || I2C1_SCL
	A5 = PA8 // ADC_IN16 || I2C1_SDA

	D0  = PC5 // USART1_RX
	D1  = PC4 // USART1_TX
	D2  = PA10
	D3  = PB3 // TIM1_CH2
	D4  = PB5
	D5  = PB4  // TIM3_CH1
	D6  = PB14 // TIM15_CH1
	D7  = PA8
	D8  = PA9
	D9  = PC7 // TIM3_CH2
	D10 = PB0 // SPI_CS || TIM3_CH3
	D11 = PA7 // SPI1_MOSI || TIM14_CH1
	D12 = PA6 // SPI1_MISO
	D13 = PA5 // SPI1_SCK
	D14 = PB9 // I2C1_SDA
	D15 = PB8 // I2C1_SCL
)

const (
	LED         = LED_BUILTIN
	LED_BUILTIN = LED_GREEN
	LED_GREEN   = LD4
	// LD4 is connected to D13/PA5, which can also be
	// configured as SPI1_SCK. Therefore the LED cannot
	// be used if SPI is used
	LD4 = PA5
)

const (
	BUTTON = PC13
)

const (
	// UART pins
	// PA2 and PA3 are connected to the ST-Link Virtual Com Port (VCP)
	UART_TX_PIN = PA2
	UART_RX_PIN = PA3

	// SPI
	SPI1_SCK_PIN = PA5
	SPI1_SDI_PIN = PA7
	SPI1_SDO_PIN = PA6
	SPI0_SCK_PIN = SPI1_SCK_PIN
	SPI0_SDI_PIN = SPI1_SDI_PIN
	SPI0_SDO_PIN = SPI1_SDO_PIN

	// I2C pins
	// PA11 and PA12 are mapped to CN4 pin 7 and CN4 pin 8 respectively
	I2C0_SCL_PIN  = PB8
	I2C0_SDA_PIN  = PB9
	I2C0_ALT_FUNC = 6
)

var (
	// USART2 is the hardware serial port connected to the onboard ST-LINK
	// debugger to be exposed as virtual COM port over USB on Nucleo boards.
	UART1  = &_UART1
	_UART1 = UART{
		Buffer:            NewRingBuffer(),
		Bus:               stm32.USART2,
		TxAltFuncSelector: 1,
		RxAltFuncSelector: 1,
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
	UART1.Interrupt = interrupt.New(stm32.IRQ_USART2, _UART1.handleInterrupt)
}
