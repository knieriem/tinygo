//go:build nucleog0b1re

package main

import "machine"

var (
	CAN_TX = machine.CAN1_TX_PIN
	CAN_RX = machine.CAN1_RX_PIN
	CAN_STANDBY = machine.NoPin
)
