//go:build stm32g0b1 || stm32h757_cm7 || stm32h723

package machine

import (
	"device/stm32"
	"errors"
	"internal/binary"
	"runtime/interrupt"
	"unsafe"
)

// This file provides a driver for the M_CAN based STM32 FDCAN
// peripherals. (As the ATSAME51/54 driver is based on M_CAN too,
// the drivers theoretically could be merged).
//
// Exported API in src/machine/can.go

// FDCAN Message RAM configuration
const (
	// Element sizes in 32-bit words
	sramcanFLSSize = 1  // Filter Standard Element Size
	sramcanFLESize = 2  // Filter Extended Element Size
	sramcanRF0Size = 18 // RX FIFO 0 Element Size (for 64-byte data)
	sramcanRF1Size = 18 // RX FIFO 1 Element Size
	sramcanRBSize  = 18 // RX Buffer Element Size
	sramcanTEFSize = 2  // TX Event FIFO Element Size
	sramcanTFQSize = 18 // TX FIFO/Queue Element Size
)

// MCANRAMConfig defines the lengths of the M_CAN Message RAM sections
type MCANRAMConfig struct {
	StdFilterLen uint8 // LSS[7:0] Max. number of standard filter elements
	ExtFilterLen uint8 // LSE[6:0] Max. number of extended filter elements

	RxFIFO0Len   uint8 // F0S[6:0] Max. number of elements in RX FIFO 0
	RxFIFO1Len   uint8 // F1S[6:0] Max. number of elements in RX FIFO 1
	RxBuffersLen uint8 // Max. number of dedicated RX Buffers

	TxEventFIFOLen uint8 // EVS[5:0] Max. number of elements in TX Event FIFO
	TxFIFOQueueLen uint8 // TQFS[5:0] Max. number of elements in TX Fifo/Queue
}

// mcanRAMOffsets contains the computed start offsets of Message RAM sections
type mcanRAMOffsets struct {
	FLSSA mcanRAMOffset
	FLESA mcanRAMOffset
	RF0SA mcanRAMOffset
	RF1SA mcanRAMOffset
	RBSA  mcanRAMOffset
	EFSA  mcanRAMOffset
	TFQSA mcanRAMOffset
}

type mcanRAMOffset uint16

func (words mcanRAMOffset) regVal() uint32 {
	return uint32(words) << 2
}
func (words mcanRAMOffset) addr() uintptr {
	return sramcanBase + uintptr(words<<2)
}
func (words mcanRAMOffset) elemAddr(index uint8, elemSize int) uintptr {
	return sramcanBase + uintptr(words.add(index, elemSize)<<2)
}
func (words mcanRAMOffset) add(n uint8, elemSize int) mcanRAMOffset {
	return words + mcanRAMOffset(n)*mcanRAMOffset(elemSize)
}

var mcanMessageRAMUsed mcanRAMOffset

func (l *MCANRAMConfig) calculateOffsets(m *mcanRAMOffsets) mcanRAMOffset {
	base := mcanMessageRAMUsed
	m.FLSSA = base
	m.FLESA = m.FLSSA.add(l.StdFilterLen, sramcanFLSSize)
	m.RF0SA = m.FLESA.add(l.ExtFilterLen, sramcanFLESize)
	m.RF1SA = m.RF0SA.add(l.RxFIFO0Len, sramcanRF0Size)
	m.RBSA = m.RF1SA.add(l.RxFIFO1Len, sramcanRF1Size)
	m.EFSA = m.RBSA.add(l.RxBuffersLen, sramcanRBSize)
	m.TFQSA = m.EFSA.add(l.TxEventFIFOLen, sramcanTEFSize)
	end := m.TFQSA.add(l.TxFIFOQueueLen, sramcanTFQSize)
	mcanMessageRAMUsed = end
	return end
}

// FDCAN element masks (for parsing message RAM)
const (
	fdcanElementMaskSTDID = 0x1FFC0000 // Standard Identifier
	fdcanElementMaskEXTID = 0x1FFFFFFF // Extended Identifier
	fdcanElementMaskRTR   = 0x20000000 // Remote Transmission Request
	fdcanElementMaskXTD   = 0x40000000 // Extended Identifier flag
	fdcanElementMaskESI   = 0x80000000 // Error State Indicator
	fdcanElementMaskTS    = 0x0000FFFF // Timestamp
	fdcanElementMaskDLC   = 0x000F0000 // Data Length Code
	fdcanElementMaskBRS   = 0x00100000 // Bit Rate Switch
	fdcanElementMaskFDF   = 0x00200000 // FD Format
	fdcanElementMaskEFC   = 0x00800000 // Event FIFO Control
	fdcanElementMaskMM    = 0xFF000000 // Message Marker
	fdcanElementMaskFIDX  = 0x7F000000 // Filter Index
	fdcanElementMaskANMF  = 0x80000000 // Accepted Non-matching Frame
)

// Interrupt flags
const (
	FDCAN_IT_RX_FIFO0_NEW_MESSAGE = 0x00000001
	FDCAN_IT_RX_FIFO0_FULL        = 0x00000002
	FDCAN_IT_RX_FIFO0_MSG_LOST    = 0x00000004
	FDCAN_IT_RX_FIFO1_NEW_MESSAGE = 0x00000010
	FDCAN_IT_RX_FIFO1_FULL        = 0x00000020
	FDCAN_IT_RX_FIFO1_MSG_LOST    = 0x00000040
	FDCAN_IT_TX_COMPLETE          = 0x00000200
	FDCAN_IT_TX_ABORT_COMPLETE    = 0x00000400
	FDCAN_IT_TX_FIFO_EMPTY        = 0x00000800
	FDCAN_IT_BUS_OFF              = 0x02000000
	FDCAN_IT_ERROR_WARNING        = 0x01000000
	FDCAN_IT_ERROR_PASSIVE        = 0x00800000
)

// Register field positions and masks.
const (
	// RXF0S.F0FL[6:0]
	mcanF0FLmask = 0x7F

	// RXF0S.F0GI[13:8]
	mcanF0GIpos  = 8
	mcanF0GImask = 0x3F

	// TXFQS.TFQPI[20:16]
	mcanTFQPIpos  = 16
	mcanTFQPImask = 0x1F

	// TXFQS.TFFL[5:0]
	mcanTFFLmask = 0x3F
)

// CAN is an STM32 FDCAN peripheral.
type CAN struct {
	Bus             *stm32.FDCAN_Type
	TxAltFuncSelect uint8
	RxAltFuncSelect uint8
	Interrupt       interrupt.Interrupt
	instance        uint8
	alwaysFD        bool
	rxInterrupt     bool
	RAMConfig       *MCANRAMConfig
	ramOffsets      mcanRAMOffsets
}

// CANTransferRate represents CAN bus transfer rates
type CANTransferRate uint32

const (
	FDCANTransferRate125kbps  CANTransferRate = 125000
	FDCANTransferRate250kbps  CANTransferRate = 250000
	FDCANTransferRate500kbps  CANTransferRate = 500000
	FDCANTransferRate1000kbps CANTransferRate = 1000000
	FDCANTransferRate2000kbps CANTransferRate = 2000000 // FD only
	FDCANTransferRate4000kbps CANTransferRate = 4000000 // FD only
)

// CANMode represents the FDCAN operating mode
type CANMode uint8

const (
	CANModeNormal           CANMode = 0
	CANModeBusMonitoring    CANMode = 1
	CANModeInternalLoopback CANMode = 2
	CANModeExternalLoopback CANMode = 3
)

// CANConfig holds FDCAN configuration parameters
type CANConfig struct {
	TransferRate      CANTransferRate // Nominal bit rate (arbitration phase)
	TransferRateFD    CANTransferRate // Data bit rate (data phase), must be >= TransferRate
	Mode              CANMode
	Tx                Pin
	Rx                Pin
	Standby           Pin  // Optional standby pin for CAN transceiver (set to NoPin if not used)
	AlwaysFD          bool // Always transmit as FD frames, even when data fits in classic CAN
	EnableRxInterrupt bool // Enable interrupt-driven receive (messages delivered via SetRxCallback)
}

// CANFilterConfig represents a message filter configuration
type CANFilterConfig struct {
	Index        uint8  // Filter index (0-27 for standard, 0-7 for extended)
	Type         uint8  // 0=Range, 1=Dual, 2=Classic (ID/Mask)
	Config       uint8  // 0=Disable, 1=FIFO0, 2=FIFO1, 3=Reject
	ID1          uint32 // First ID or filter
	ID2          uint32 // Second ID or mask
	IsExtendedID bool   // true for 29-bit ID, false for 11-bit
}

var (
	errCANInvalidTransferRate   = errors.New("CAN: invalid TransferRate")
	errCANInvalidTransferRateFD = errors.New("CAN: invalid TransferRateFD")
	errCANTimeout               = errors.New("CAN: timeout")
	errCANTxFifoFull            = errors.New("CAN: Tx FIFO full")
	errCANMissingRAMConfig      = errors.New("CAN: missing Message RAM config")
)

// flags implemented as described in [CAN.SetRxCallback]
var canRxCB [numFDCANInstances]canRxCallback

// canInstances tracks CAN peripherals with interrupt-driven RX enabled.
// A non-nil entry means setRxCallback was called with a non-nil callback.
var canInstances [numFDCANInstances]*CAN

// Configure initializes the FDCAN peripheral and starts it.
func (can *CAN) Configure(config CANConfig) error {
	can.alwaysFD = config.AlwaysFD

	if config.Standby != NoPin {
		config.Standby.Configure(PinConfig{Mode: PinOutput})
		config.Standby.Low()
	}

	enableFDCANClock()

	config.Tx.ConfigureAltFunc(PinConfig{Mode: PinModeFDCANTx}, can.TxAltFuncSelect)
	config.Rx.ConfigureAltFunc(PinConfig{Mode: PinModeFDCANRx}, can.RxAltFuncSelect)

	// Exit sleep mode.
	can.Bus.SetCCCR_CSR(0)
	timeout := 10000
	for can.Bus.GetCCCR_CSA() != 0 {
		timeout--
		if timeout == 0 {
			return errCANTimeout
		}
	}

	// Request initialization.
	can.Bus.SetCCCR_INIT(1)
	timeout = 10000
	for can.Bus.GetCCCR_INIT() == 0 {
		timeout--
		if timeout == 0 {
			return errCANTimeout
		}
	}

	// Enable configuration change.
	can.Bus.SetCCCR_CCE(1)

	can.setClockDiv()

	can.Bus.SetCCCR_DAR(0)  // Enable auto retransmission.
	can.Bus.SetCCCR_TXP(0)  // Disable transmit pause.
	can.Bus.SetCCCR_PXHD(0) // Enable protocol exception handling.
	can.Bus.SetCCCR_FDOE(1) // FD operation.
	can.setCCCR_BRSE(1)     // Bit rate switching.

	// Reset mode bits, then apply requested mode.
	can.Bus.SetCCCR_TEST(0)
	can.Bus.SetCCCR_MON(0)
	can.Bus.SetCCCR_ASM(0)
	can.Bus.SetTEST_LBCK(0)
	switch config.Mode {
	case CANModeBusMonitoring:
		can.Bus.SetCCCR_MON(1)
	case CANModeInternalLoopback:
		can.Bus.SetCCCR_TEST(1)
		can.Bus.SetCCCR_MON(1)
		can.Bus.SetTEST_LBCK(1)
	case CANModeExternalLoopback:
		can.Bus.SetCCCR_TEST(1)
		can.Bus.SetTEST_LBCK(1)
	}

	// Nominal bit timing (64 MHz FDCAN clock, 16 tq/bit, ~80% sample point).
	if config.TransferRate == 0 {
		config.TransferRate = FDCANTransferRate500kbps
	}
	nbrp, ntseg1, ntseg2, nsjw, err := fdcanNominalBitTiming(config.TransferRate)
	if err != nil {
		return err
	}
	can.Bus.NBTP.Set(((nsjw - 1) << 25) | ((nbrp - 1) << 16) | ((ntseg1 - 1) << 8) | (ntseg2 - 1))

	// Data bit timing (FD phase).
	if config.TransferRateFD == 0 {
		config.TransferRateFD = FDCANTransferRate1000kbps
	}
	if config.TransferRateFD < config.TransferRate {
		return errCANInvalidTransferRateFD
	}
	dbrp, dtseg1, dtseg2, dsjw, err := fdcanDataBitTiming(config.TransferRateFD)
	if err != nil {
		return err
	}
	can.Bus.DBTP.Set(((dbrp - 1) << 16) | ((dtseg1 - 1) << 8) | ((dtseg2 - 1) << 4) | (dsjw - 1))

	// Enable timestamp counter (internal, prescaler=1).
	can.Bus.TSCC.Set(1)

	// Setup Message RAM offsets.
	err = can.ensureRAMConfig()
	if err != nil {
		return err
	}
	size := can.RAMConfig.calculateOffsets(&can.ramOffsets)

	// Clear message RAM.
	base := sramcanBase + uintptr(can.ramOffsets.FLSSA<<2)
	end := sramcanBase + uintptr(size<<2)
	for addr := base; addr < end; addr += 4 {
		*(*uint32)(unsafe.Pointer(addr)) = 0
	}

	can.configFilterGlobal()
	can.configMessageRAM()

	// Start peripheral.
	can.Bus.SetCCCR_CCE(0)
	can.Bus.SetCCCR_INIT(0)
	timeout = 10000
	for can.Bus.GetCCCR_INIT() != 0 {
		timeout--
		if timeout == 0 {
			return errCANTimeout
		}
	}

	return nil
}

// Stop puts the FDCAN peripheral back into initialization mode.
func (can *CAN) Stop() error {
	can.Bus.SetCCCR_INIT(1)
	timeout := 10000
	for can.Bus.GetCCCR_INIT() == 0 {
		timeout--
		if timeout == 0 {
			return errCANTimeout
		}
	}
	can.Bus.SetCCCR_CCE(1)
	return nil
}

// txFIFOLevel implements [CAN.TxFIFOLevel].
func (can *CAN) txFIFOLevel() (int, int) {
	max := int(can.RAMConfig.TxFIFOQueueLen)
	free := int(can.Bus.TXFQS.Get() & mcanTFFLmask)
	return max - free, max
}

// tx implements [CAN.Tx].
func (can *CAN) tx(id canID, flags canFlags, data []byte) error {
	if can.Bus.TXFQS.Get()&0x00200000 != 0 { // TFQF bit
		return errCANTxFifoFull
	}

	length := byte(len(data))
	if length > 64 {
		length = 64
	}

	// Use FD framing if configured to always use FD, or if data exceeds classic CAN max.
	isFD := flags&canFlagFDF != 0 || length > 8

	putIndex := (can.Bus.TXFQS.Get() >> mcanTFQPIpos) & mcanTFQPImask
	txAddr := can.ramOffsets.TFQSA.elemAddr(uint8(putIndex), sramcanTFQSize)

	// Header word 1: identifier and flags.
	var w1 uint32
	if flags&canFlagIDE != 0 {
		w1 = (id & 0x1FFFFFFF) | fdcanElementMaskXTD
	} else {
		w1 = (id & 0x7FF) << 18
	}

	// Header word 2: DLC, FD/BRS flags.
	dlc := lengthToDLC(length)
	w2 := uint32(dlc) << 16
	if isFD {
		w2 |= fdcanElementMaskFDF | fdcanElementMaskBRS
	}

	*(*uint32)(unsafe.Pointer(txAddr)) = w1
	*(*uint32)(unsafe.Pointer(txAddr + 4)) = w2

	copyToBuffer(txAddr, data[:length])
	can.Bus.TXBAR.Set(1 << putIndex)
	return nil
}

func copyToBuffer(txAddr uintptr, data []byte) {
	n := len(data)
	fullWords := n / 4
	for w := range fullWords {
		word := binary.LittleEndian.Uint32(data[w*4:])
		*(*uint32)(unsafe.Pointer(txAddr + 8 + uintptr(w)*4)) = word
	}
	if remainder := n - fullWords*4; remainder > 0 {
		var tail [4]byte
		copy(tail[:], data[fullWords*4:n])
		word := binary.LittleEndian.Uint32(tail[:])
		*(*uint32)(unsafe.Pointer(txAddr + 8 + uintptr(fullWords)*4)) = word
	}
}

// rxFIFOLevel implements [CAN.RxFIFOLevel].
// Returns 0,0 when interrupt-driven (messages delivered via callback).
func (can *CAN) rxFIFOLevel() (int, int) {
	if canInstances[can.instance] != nil {
		return 0, 0
	}
	level := int(can.Bus.RXF0S.Get() & mcanF0FLmask)
	return level, int(can.RAMConfig.RxFIFO0Len)
}

// setRxCallback implements [CAN.SetRxCallback].
// When cb is non-nil, interrupt-driven receive is enabled on RX FIFO 0.
// The CAN.Interrupt field must be initialized with interrupt.New in the board file.
func (can *CAN) setRxCallback(cb canRxCallback) {
	canRxCB[can.instance] = cb
	if cb != nil {
		canInstances[can.instance] = can
		// Enable RX FIFO 0 new message interrupt, routed to interrupt line 0.
		can.Bus.SetIE_RF0NE(1)
		can.setILS_RF0NL(0)
		can.Bus.SetILE_EINT0(1)
		can.Interrupt.Enable()
	} else {
		can.Bus.SetIE_RF0NE(0)
		canInstances[can.instance] = nil
	}
}

// rxPoll implements [CAN.RxPoll].
// No-op when interrupt-driven receive is active.
func (can *CAN) rxPoll() error {
	if canInstances[can.instance] != nil {
		// FIXME: disable return to make RxPoll work.
		// If can.SetRxCallback is called, canInstances is always set,
		// also if the intention is to use RxPoll, not interrupts.
		if false {
			return nil
		}
	}
	cb := canRxCB[can.instance]
	if cb == nil {
		return nil
	}
	processRxFIFO0(can, cb)
	return nil
}

// processRxFIFO0 drains RX FIFO 0 and delivers each message to cb.
// Used by both rxPoll (poll mode) and canHandleInterrupt (interrupt mode).
func processRxFIFO0(can *CAN, cb canRxCallback) {
	for can.Bus.RXF0S.Get()&mcanF0FLmask != 0 {
		getIndex := (can.Bus.RXF0S.Get() >> mcanF0GIpos) & mcanF0GImask
		rxAddr := can.ramOffsets.RF0SA.elemAddr(uint8(getIndex), sramcanRF0Size)

		w1 := *(*uint32)(unsafe.Pointer(rxAddr))
		w2 := *(*uint32)(unsafe.Pointer(rxAddr + 4))

		extendedID := w1&fdcanElementMaskXTD != 0
		var id uint32
		var flags uint32
		if extendedID {
			flags |= canFlagIDE
			id = w1 & fdcanElementMaskEXTID
		} else {
			id = (w1 & fdcanElementMaskSTDID) >> 18
		}

		timestamp := w2 & fdcanElementMaskTS
		dlc := byte((w2 & fdcanElementMaskDLC) >> 16)
		isFD := w2&fdcanElementMaskFDF != 0

		if isFD {
			flags |= canFlagFDF
		}
		if w1&fdcanElementMaskRTR != 0 {
			flags |= canFlagRTR
		}
		if w2&fdcanElementMaskBRS != 0 {
			flags |= canFlagBRS
		}
		if w1&fdcanElementMaskESI != 0 {
			flags |= canFlagESI
		}

		dataLen := dlcToLength(dlc)
		if !isFD && dataLen > 8 {
			dataLen = 8
		}
		var buf [64]byte
		for w := range (dataLen + 3) / 4 {
			word := *(*uint32)(unsafe.Pointer(rxAddr + 8 + uintptr(w)*4))
			binary.LittleEndian.PutUint32(buf[w*4:], word)
		}

		// Acknowledge before callback so the FIFO slot is freed.
		can.Bus.RXF0A.Set(uint32(getIndex))
		cb(buf[:dataLen], id, timestamp, flags)
	}
}

// canHandleInterrupt is the shared interrupt handler for FDCAN interrupt line 0 (IRQ_TIM16).
// Both FDCAN1 and FDCAN2 share this IRQ vector.
func canHandleInterrupt(interrupt.Interrupt) {
	for i := range canInstances {
		can := canInstances[i]
		if can == nil {
			continue
		}
		ir := can.Bus.IR.Get()
		if ir&FDCAN_IT_RX_FIFO0_NEW_MESSAGE != 0 {
			can.Bus.IR.Set(FDCAN_IT_RX_FIFO0_NEW_MESSAGE) // Write 1 to clear
			if cb := canRxCB[i]; cb != nil {
				processRxFIFO0(can, cb)
			}
		}
	}
}

// ConfigureFilter configures a message acceptance filter.
func (can *CAN) ConfigureFilter(config CANFilterConfig) error {
	if config.IsExtendedID {
		if config.Index >= can.RAMConfig.ExtFilterLen {
			return errors.New("CAN: filter index out of range")
		}

		filterAddr := can.ramOffsets.FLESA.elemAddr(config.Index, sramcanFLESize)

		w1 := (uint32(config.Config) << 29) | (config.ID1 & 0x1FFFFFFF)
		w2 := (uint32(config.Type) << 30) | (config.ID2 & 0x1FFFFFFF)

		*(*uint32)(unsafe.Pointer(filterAddr)) = w1
		*(*uint32)(unsafe.Pointer(filterAddr + 4)) = w2
	} else {
		if config.Index >= can.RAMConfig.StdFilterLen {
			return errors.New("CAN: filter index out of range")
		}

		filterAddr := can.ramOffsets.FLSSA.elemAddr(config.Index, sramcanFLSSize)

		w := (uint32(config.Type) << 30) |
			(uint32(config.Config) << 27) |
			((config.ID1 & 0x7FF) << 16) |
			(config.ID2 & 0x7FF)

		*(*uint32)(unsafe.Pointer(filterAddr)) = w
	}

	return nil
}

// fdcanNominalBitTiming returns prescaler and segment values for the nominal (arbitration) phase.
// STM32G0 FDCAN clock = 64 MHz, 16 time quanta per bit, ~80% sample point.
func fdcanNominalBitTiming(rate CANTransferRate) (brp, tseg1, tseg2, sjw uint32, err error) {
	switch rate {
	case FDCANTransferRate125kbps:
		return 32, 13, 2, 4, nil
	case FDCANTransferRate250kbps:
		return 16, 13, 2, 4, nil
	case FDCANTransferRate500kbps:
		return 8, 13, 2, 4, nil
	case FDCANTransferRate1000kbps:
		return 4, 13, 2, 4, nil
	default:
		return 0, 0, 0, 0, errCANInvalidTransferRate
	}
}

// fdcanDataBitTiming returns prescaler and segment values for the data phase (FD).
func fdcanDataBitTiming(rate CANTransferRate) (brp, tseg1, tseg2, sjw uint32, err error) {
	switch rate {
	case FDCANTransferRate125kbps:
		return 32, 13, 2, 4, nil
	case FDCANTransferRate250kbps:
		return 16, 13, 2, 4, nil
	case FDCANTransferRate500kbps:
		return 8, 13, 2, 4, nil
	case FDCANTransferRate1000kbps:
		return 4, 13, 2, 4, nil
	case FDCANTransferRate2000kbps:
		return 2, 13, 2, 4, nil
	case FDCANTransferRate4000kbps:
		return 1, 13, 2, 4, nil
	default:
		return 0, 0, 0, 0, errCANInvalidTransferRateFD
	}
}
