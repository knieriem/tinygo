//go:build nucleoh723zg

package main

import "machine"

var (
	CAN_TX = machine.CAN1_TX
	CAN_RX = machine.CAN1_RX
	CAN_STANDBY = machine.NoPin
)
