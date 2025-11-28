
MEMORY
{
  FLASH_TEXT (rx) : ORIGIN = 0x08000000, LENGTH = 1024K
  RAM (xrw)       : ORIGIN = 0x24000000, LENGTH = 128K
  D2_AHB_SRAM1 (rw)	: ORIGIN = 0x30000000, LENGTH = 16K
}

_stack_size = 4K;

INCLUDE "targets/arm.ld"

/* define output sections */
SECTIONS
{
    .ram_nocache (NOLOAD):
    {
        . = ALIGN(4);
        *(.ram_nocache)
        *(.ram_nocache*)
        . = ALIGN(4);
    } >D2_AHB_SRAM1
}
