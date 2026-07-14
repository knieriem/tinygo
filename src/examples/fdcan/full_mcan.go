//go:build stm32h723 || stm32h757_cm7

package main

import (
	"machine"
)

var messageRAMConfig = machine.MCANRAMConfig{
	StdFilterLen:   28,
	ExtFilterLen:   8,
	RxFIFO0Len:     16,
	RxFIFO1Len:     0,
	TxEventFIFOLen: 0,
	TxFIFOQueueLen: 16,
}

func init() {
	machine.CAN1.RAMConfig = &messageRAMConfig
}
