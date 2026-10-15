package main

// This is a CAN example using the new CAN API. It should run on
// STM32 boards like Nucleo G0B1RE and H723ZG.

import (
	"device/stm32"
	"machine"
	"time"

	"github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing"
	"github.com/tinygo-org/tinygo/src/examples/fdcan/internal/timing/dev"
)

const fCAN = 64e6

// Bit timing calculation (likely performed at compile time).
var nominalBT = bitTiming(250e3, 800, &dev.MCAN.Nominal)
var dataBT = bitTiming(250e3, 700, dev.MCAN.Data)

var led = machine.LED

func main() {
	led.Configure(machine.PinConfig{Mode: machine.PinOutput})

	can1 := machine.CAN1
	err := can1.Configure(machine.CANConfig{
		Nominal: nominalBT,
		Data:    dataBT,
		Rx:      CAN_RX,
		Tx:      CAN_TX,
		Standby: CAN_STANDBY,
	})

	if err != nil {
		signalError()
	}

	can1.SetRxCallback(func(data []byte, id uint32, timestamp, flags uint32) {
		// send each received message back, with an ID incremented by one
		id++
		can1.Tx(id, flags, data)
		pulseLED()
	})

	// Send a message each 0.5s, incrementing its length up to 8,
	// then starting at zero again.
	txData := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}
	n := 0
	for {
		can1.RxPoll()
		if !toggleLED() {
			time.Sleep(time.Second / 4)
			continue
		}

		failed := ""
		err := can1.Tx(0x765, 0, txData[:n])
		if err != nil {
			failed = "failed"
		}
		n++
		if n == 8 {
			n = 0
		}
		println(stm32.FDCAN1.TXBC.Get(), stm32.FDCAN1.PSR.Get(), stm32.FDCAN1.RXF0S.Get()&0x7F, failed)
		time.Sleep(time.Second / 4)
		machine.Watchdog.Update()
	}
}

func bitTiming(bitrate uint32, sample1000 int, c *timing.Constraints) machine.CANBitTiming {
	var bt timing.BitTiming
	err := timing.CalcBitTiming(&bt, fCAN, bitrate, timing.Frac1000(sample1000), c)
	if err == nil {
		bt.EnsureSJW(c.SJWMax, 0)
	}
	// if CalcBitTiming failed, it won't have modified bt, so can.Configure
	// will detect invalid bit timing and return an error.
	return machine.CANBitTiming{
		Prescaler: bt.Prescaler,
		TSeg1:     bt.TSeg1,
		TSeg2:     bt.TSeg2,
		SJW:       bt.SJW,
	}
}

var ledOn bool
var ledPulse bool
var ledTicks int
var pulseEnd int

func toggleLED() bool {
	ledOn = !ledOn
	ledTicks++
	if ledPulse {
		if ledTicks-pulseEnd < 0 {
			return ledOn
		}
		ledPulse = false
	}
	if ledOn {
		led.Low()
	} else {
		led.High()
	}
	return ledOn
}

func pulseLED() {
	led.Low()
	ledPulse = true
	pulseEnd = ledTicks + 3
}

func signalError() {
	for {
		toggleLED()
		time.Sleep(time.Millisecond * 20)
		machine.Watchdog.Update()
	}
}
