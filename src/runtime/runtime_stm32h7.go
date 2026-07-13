//go:build stm32 && (stm32h723 || stm32h757_cm7)

package runtime

import (
	"device/arm"
	"device/stm32"
	"machine"
	"runtime/volatile"
)

/*
clock settings

	+-------------+-----------+
	| HSI         |  64 MHz   |
	| SYSCLK      | 192 MHz   | = HSI/1 / M * N / P
	| HCLK        |  96 MHz   | = SYSCLK / HPRE
	| APB1(PCLK1) |  48 MHz   | = HCLK / 2
	| APB2(PCLK2) |  48 MHz   | = HCLK / 2
	| APB3(PCLK3) |  48 MHz   | = HCLK / 2
	| APB4(PCLK4) |  48 MHz   | = HCLK / 2
	| PLL1_Q      |  64 MHz   | = HSI/1 / M * N / Q  (FDCAN kernel clock)
	+-------------+-----------+
*/

const (
	HSE_STARTUP_TIMEOUT = 0x0500

	voltageScale2 = 0b10

	PLL_M = 4
	PLL_N = 12
	PLL_P = 1
	PLL_Q = 3 // for FDCAN -> 192 MHz / 3 = 64 MHz
	PLL_R = 2

	PLL_RGE    = stm32.RCC_PLLCFGR_PLL1RGE_Range8
	PLL_VCOSEL = stm32.RCC_PLLCFGR_PLL1VCOSEL_WideVCO
	PLL_FRACN  = 0

	SysclkDiv = 0

	// Core and bus clock dividers,
	// see [RM0468, p. 332, fig. 49: Core and bus clock generation][clkgen]
	//
	// [clkgen]: https://www.st.com/resource/en/reference_manual/rm0468-stm32h723733-stm32h725735-and-stm32h730-value-line-advanced-armbased-32bit-mcus-stmicroelectronics.pdf#page=332
	HPREDiv = stm32.RCC_D1CFGR_HPRE_Div2
	APB1Div = stm32.RCC_D2CFGR_D2PPRE1_Div2
	APB2Div = stm32.RCC_D2CFGR_D2PPRE2_Div2
	APB3Div = stm32.RCC_D1CFGR_D1PPRE_Div2
	APB4Div = stm32.RCC_D3CFGR_D3PPRE_Div2

	FlashLatency = 1
)

func init() {
	initMPU()
	initCLK()

	machine.InitSerial()

	initTickTimer(&machine.TIM2)
}

func putchar(c byte) {
	machine.Serial.WriteByte(c)
}

func getchar() byte {
	for machine.Serial.Buffered() == 0 {
		Gosched()
	}
	v, _ := machine.Serial.ReadByte()
	return v
}

func buffered() int {
	return machine.Serial.Buffered()
}

func initMPU() {
	arm.Asm("dmb 0xF")

	// Disable memory fault exceptions
	arm.SCB.SHCSR.ClearBits(arm.SCB_SHCSR_MEMFAULTENA)

	// disable the MPU
	arm.MPU.CTRL.Set(0)

	arm.MPU.RNR.Set(0)

	arm.MPU.RASR.ClearBits(arm.MPU_RASR_ENABLE)

	arm.MPU.RBAR.Set(0x3000_0000)
	arm.MPU.RASR.Set(
		0b000<<arm.MPU_RASR_TEX_Pos | // TEX: Level 0
			0b0<<arm.MPU_RASR_B_Pos | // Not bufferable
			0b0<<arm.MPU_RASR_C_Pos | // Not cacheable
			0b0<<arm.MPU_RASR_S_Pos | // Not shareable
			0b011<<arm.MPU_RASR_AP_Pos | // Full access
			0<<arm.MPU_RASR_XN_Pos | // Executable
			13<<arm.MPU_RASR_SIZE_Pos | // 16kb
			0x00<<arm.MPU_RASR_SRD_Pos | // Subregions enabled
			1<<arm.MPU_RASR_ENABLE_Pos) // Region enable

	arm.SCB.SHCSR.SetBits(arm.SCB_SHCSR_MEMFAULTENA) // enable memory faults
	arm.MPU.CTRL.SetBits(arm.MPU_CTRL_PRIVDEFENA | arm.MPU_CTRL_ENABLE)
	arm.Asm("dsb 0xF")
	arm.Asm("isb 0xF")
}

func initCLK() {
	configPWRSupply(stm32.PWR_CR3_LDOEN)

	enableHSI(stm32.RCC_CR_HSIDIV_Div1, 64)

	// Initialize the High-Speed External Oscillator
	//	initOsc()

	setFlashLatency(FlashLatency, ifPrevLower)
	selectVoltageScaling(voltageScale2, ifPrevHigher)

	setSysclkSrc(stm32.RCC_CFGR_SW_HSI)

	setAPB3Div(APB3Div, ifPrevLower)
	setAPB1Div(APB1Div, ifPrevLower)
	setAPB2Div(APB2Div, ifPrevLower)
	setAPB4Div(APB4Div, ifPrevLower)
	setHPREDiv(HPREDiv, ifPrevLower)

	initPLL1()
	setSysclkDiv(SysclkDiv)

	setSysclkSrc(stm32.RCC_CFGR_SW_PLL1)

	setHPREDiv(HPREDiv, ifPrevHigher)
	setAPB3Div(APB3Div, ifPrevHigher)
	setAPB1Div(APB1Div, ifPrevHigher)
	setAPB2Div(APB2Div, ifPrevHigher)
	setAPB4Div(APB4Div, ifPrevHigher)
	selectVoltageScaling(voltageScale2, ifPrevLower)
	setFlashLatency(FlashLatency, ifPrevHigher)
}

func initPLL1() {
	stm32.RCC.CR.ClearBits(stm32.RCC_CR_PLL1ON)
	for stm32.RCC.CR.HasBits(stm32.RCC_CR_PLL1RDY) {
	}

	stm32.RCC.PLLCKSELR.ReplaceBits(
		stm32.RCC_PLLCKSELR_PLLSRC_HSI<<stm32.RCC_PLLCKSELR_PLLSRC_Pos|
			uint32(PLL_M)<<stm32.RCC_PLLCKSELR_DIVM1_Pos,
		stm32.RCC_PLLCKSELR_PLLSRC_Msk|stm32.RCC_PLLCKSELR_DIVM1_Msk, 0)

	stm32.RCC.PLLCFGR.ReplaceBits(
		PLL_RGE<<stm32.RCC_PLLCFGR_PLL1RGE_Pos|
			PLL_VCOSEL<<stm32.RCC_PLLCFGR_PLL1VCOSEL_Pos|
			stm32.RCC_PLLCFGR_DIVP1EN|
			stm32.RCC_PLLCFGR_DIVQ1EN|
			stm32.RCC_PLLCFGR_DIVR1EN,
		stm32.RCC_PLLCFGR_PLL1RGE_Msk|stm32.RCC_PLLCFGR_PLL1VCOSEL_Msk|
			stm32.RCC_PLLCFGR_DIVP1EN|stm32.RCC_PLLCFGR_DIVQ1EN|stm32.RCC_PLLCFGR_DIVR1EN, 0)

	stm32.RCC.PLL1DIVR.Set(
		uint32(PLL_N-1)<<stm32.RCC_PLL1DIVR_DIVN1_Pos |
			uint32(PLL_P-1)<<stm32.RCC_PLL1DIVR_DIVP1_Pos |
			uint32(PLL_Q-1)<<stm32.RCC_PLL1DIVR_DIVQ1_Pos |
			uint32(PLL_R-1)<<stm32.RCC_PLL1DIVR_DIVR1_Pos)

	stm32.RCC.PLL1FRACR.Set(uint32(PLL_FRACN) << stm32.RCC_PLL1FRACR_FRACN1_Pos)

	stm32.RCC.CR.SetBits(stm32.RCC_CR_PLL1ON)
	for !stm32.RCC.CR.HasBits(stm32.RCC_CR_PLL1RDY) {
	}
}

func selectVoltageScaling(scaleSpec uint32, allowed func(want, have uint32) bool) {

	cur := (stm32.PWR.D3CR.Get() & stm32.PWR_D3CR_VOS_Msk) >> stm32.PWR_D3CR_VOS_Pos
	if !allowed(scaleSpec, cur) {
		return
	}
	stm32.PWR.D3CR.ReplaceBits(scaleSpec<<stm32.PWR_D3CR_VOS_Pos, stm32.PWR_D3CR_VOS_Msk, 0)
	_ = stm32.PWR.D3CR.Get()

	// FIXME: H723 does not leave the loop if it is active;
	// on H757_CM7, it works fine. Needs investigation.
	// for !stm32.PWR.D3CR.HasBits(stm32.PWR_D3CR_VOSRDY) {
	// }
}

func configPWRSupply(config uint32) {
	const msk = stm32.PWR_CR3_SDEN | stm32.PWR_CR3_LDOEN | stm32.PWR_CR3_BYPASS

	// If the step-down converter is already enabled, do not change the settings.
	// Not sure if this is necessary, but CubeMX appears to do it that way.
	if stm32.PWR.CR3.HasBits(stm32.PWR_CR3_SDEN) {
		return
	}
	stm32.PWR.CR3.ReplaceBits(config, msk, 0)

	for !stm32.PWR.CSR1.HasBits(stm32.PWR_CSR1_ACTVOSRDY) {
	}
}

func setSysclkSrc(sel uint32) {
	stm32.RCC.CFGR.ReplaceBits(sel<<stm32.RCC_CFGR_SW_Pos, stm32.RCC_CFGR_SW_Msk, 0)
	for (stm32.RCC.CFGR.Get()&stm32.RCC_CFGR_SWS_Msk)>>stm32.RCC_CFGR_SWS_Pos != sel {
	}
}

func enableHSI(div uint, calibVal uint8) {
	sysclkSrc := (stm32.RCC.CFGR.Get() & stm32.RCC_CFGR_SWS_Msk) >> stm32.RCC_CFGR_SWS_Pos
	pllclkSrc := (stm32.RCC.PLLCKSELR.Get() & stm32.RCC_PLLCKSELR_PLLSRC_Msk) >> stm32.RCC_PLLCKSELR_PLLSRC_Pos
	if sysclkSrc == stm32.RCC_CFGR_SWS_HSI ||
		(sysclkSrc == stm32.RCC_CFGR_SWS_PLL1 && pllclkSrc == stm32.RCC_PLLCKSELR_PLLSRC_HSI) {
		// HSI is already in use, configure only

	}
	stm32.RCC.CR.ReplaceBits(
		stm32.RCC_CR_HSION<<stm32.RCC_CR_HSION_Pos|
			uint32(div)<<stm32.RCC_CR_HSIDIV_Pos,
		stm32.RCC_CR_HSION_Msk|stm32.RCC_CR_HSIDIV_Msk, 0)

	for !stm32.RCC.CR.HasBits(stm32.RCC_CR_HSIRDY_Msk) {
	}

	stm32.RCC.HSICFGR.ReplaceBits(uint32(calibVal)<<stm32.RCC_HSICFGR_HSITRIM_Pos, stm32.RCC_HSICFGR_HSITRIM_Msk, 0)
}

// setHPREDiv configures the HCLK pre-divider (HPRE), if the condition is met.
// HCLK feeds the AHB
func setHPREDiv(div uint32, allowed func(want, have uint32) bool) {
	setD1Div(div<<stm32.RCC_D1CFGR_HPRE_Pos, stm32.RCC_D1CFGR_HPRE_Msk, allowed)
}

func setSysclkDiv(div uint32) {
	setD1Div(div<<stm32.RCC_D1CFGR_D1CPRE_Pos, stm32.RCC_D1CFGR_D1CPRE_Msk, nil)
}

func setAPB1Div(div uint32, allowed func(want, have uint32) bool) {
	setD2Div(div<<stm32.RCC_D2CFGR_D2PPRE1_Pos, stm32.RCC_D2CFGR_D2PPRE1_Msk, allowed)
}

func setAPB2Div(div uint32, allowed func(want, have uint32) bool) {
	setD2Div(div<<stm32.RCC_D2CFGR_D2PPRE2_Pos, stm32.RCC_D2CFGR_D2PPRE2_Msk, allowed)
}

func setAPB3Div(div uint32, allowed func(want, have uint32) bool) {
	setDiv(&stm32.RCC.D1CFGR, div<<stm32.RCC_D1CFGR_D1PPRE_Pos, stm32.RCC_D1CFGR_D1PPRE_Msk, allowed)
}

func setAPB4Div(div uint32, allowed func(want, have uint32) bool) {
	setDiv(&stm32.RCC.D3CFGR, div<<stm32.RCC_D3CFGR_D3PPRE_Pos, stm32.RCC_D3CFGR_D3PPRE_Msk, allowed)
}

func setD1Div(div, mask uint32, allowed func(want, have uint32) bool) {
	setDiv(&stm32.RCC.D1CFGR, div, mask, allowed)
}

func setD2Div(div, mask uint32, allowed func(want, have uint32) bool) {
	setDiv(&stm32.RCC.D2CFGR, div, mask, allowed)
}

func setDiv(r *volatile.Register32, div, mask uint32, allowed func(want, have uint32) bool) {
	curDiv := r.Get() & mask
	if allowed != nil && !allowed(div, curDiv) {
		return
	}
	r.ReplaceBits(div, mask, 0)
}

func setFlashLatency(latency uint32, allowed func(want, have uint32) bool) {
	cur := (stm32.FLASH.ACR.Get() & stm32.FLASH_ACR_LATENCY_Msk) >> stm32.FLASH_ACR_LATENCY_Pos
	if !allowed(latency, cur) {
		return
	}
	stm32.FLASH.ACR.ReplaceBits(latency<<stm32.FLASH_ACR_LATENCY_Pos, stm32.FLASH_ACR_LATENCY_Msk, 0)
	_ = stm32.FLASH.ACR.Get()
}

func ifPrevLower(next, prev uint32) bool {
	return next > prev
}

func ifPrevHigher(next, prev uint32) bool {
	return next < prev
}
