//go:build amken_trio

package main

import "machine"

var (
	CAN_TX = machine.CAN_TX
	CAN_RX = machine.CAN_RX
	CAN_STANDBY = machine.NoPin
)
