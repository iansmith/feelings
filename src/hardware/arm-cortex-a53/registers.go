package arm_cortex_a53

// ***************************************
// SCTLR_EL1, System Control Register (EL1), Page 2654 of AArch64-Reference-Manual.
// ***************************************

const SystemControlRegisterEL1Reserved = (3 << 28) | (3 << 22) | (1 << 20) | (1 << 11)
const SystemControlRegisterEELittleEndian = (0 << 25)
const SystemControlRegisterEOELittleEndian = (0 << 24)
const SystemControlRegisterICacheDisabled = (0 << 12)
const SystemControlRegisterDCacheDisabled = (0 << 2)
const SystemControlRegisterMMUDisabled = (0 << 0)
const SystemControlRegisterMMUEnabled = (1 << 0)

const SystemControlRegisterValueMMUDisabled = (SystemControlRegisterEL1Reserved | // 0x30000000 | 0xC00000 | 0x100000 | 0x800 => 0x30D00800
	SystemControlRegisterEELittleEndian | //0x0
	SystemControlRegisterICacheDisabled | //0x0
	SystemControlRegisterDCacheDisabled | //0x0
	SystemControlRegisterMMUDisabled) //0x0

// ***************************************
// HCR_EL2, Hypervisor Configuration Register (EL2), Page 2487 of AArch64-Reference-Manual.
// ***************************************

const HypervisorConfigurationRegisterRW = (1 << 31)
const HypervisorConfigurationRegisterValue = HypervisorConfigurationRegisterRW //0x80000000

// ***************************************
// SCR_EL3, Secure Configuration Register (EL3), Page 2648 of AArch64-Reference-Manual.
// ***************************************

const SecureConfigurationRegisterReserved = (3 << 4)
const SecureConfigurationRegisterRW = (1 << 10)
const SecureConfigurationRegisterNS = (1 << 0)
const SecureConfigurationRegisterValue = //0x30 | 0x400 | 0x1 => 0x431
SecureConfigurationRegisterReserved |
	SecureConfigurationRegisterRW |
	SecureConfigurationRegisterNS

// ***************************************
// SPSR_EL3, Saved Program Status Register (EL3) Page 389 of AArch64-Reference-Manual.
// ***************************************

const SavedProgramStatusRegisterMaskAll = (7 << 6)
const SavedProgramStatusRegisterEl1h = (5 << 0) //EL1 has own stack
const SavedProgramStatusRegisterValue = SavedProgramStatusRegisterMaskAll |
	SavedProgramStatusRegisterEl1h //0x1C0
