package bcm2835

import "github.com/tinygo-org/tinygo/src/runtime/volatile"

type EMCCRegisterMap struct {
	Arg2                volatile.Register32 // 0x00
	BlockSizAndCount    volatile.Register32 // 0x04
	Arg1                volatile.Register32 // 0x08
	CommandTransferMode volatile.Register32 // 0x0C
	Response0           volatile.Register32 //0x10
	Response1           volatile.Register32 //0x14
	Response2           volatile.Register32 //0x18
	Response3           volatile.Register32 //0x1C
	Data                volatile.Register32 //0x20
	Status              volatile.Register32 //0x24
	Control0            volatile.Register32 //0x28 //host config bits
	Control1            volatile.Register32 //0x2C //host config bits
	Interrupt           volatile.Register32 //0x30 //intr flags
	InterruptMask       volatile.Register32 //0x34 //intr enable
	InterruptEnable     volatile.Register32 //0x38 //intr generat
	Control2            volatile.Register32 //0x3C //host config bits
	reserved0           [4]uint32
	ForceInterruptEvent volatile.Register32 //0x50
	reserved1           [7]uint32
	BootTimeout         volatile.Register32 //0x70
	DebugBusConfig      volatile.Register32 //0x74
	reserved2           [2]uint32
	ExtensionFIFOConfig volatile.Register32 //0x80
	ExtensionFIFOEnable volatile.Register32 //0x84
	TuneStep            volatile.Register32 //0x88 //delay per card clock tuning step
	TuneStepsSDR        volatile.Register32 //0x8C
	TuneStepsDDR        volatile.Register32 //0x90
	reserved3           [3]uint32           //to 0xA0
	reserved4           [20]uint32          //to 0xFO
	SPIInterruptSupport volatile.Register32 //0xf0
	reserved5           [2]uint32
	SlotInterruptStatus volatile.Register32 //0xfc
}

type SDInitFailure struct {
	msg string
}

func NewSDInitFailure(s string) error {
	return &SDInitFailure{
		msg: s,
	}
}
func (s *SDInitFailure) Error() string {
	return s.msg
}

// SLOTISR_VER values
const HostSpecNum = 0x00ff0000
const HostSpecNumShift = 16
const HostSpecV3 = 2
const HostSpecV2 = 1
const HostSpecV1 = 0

// CONTROL register 0 settings
const Control0SPIModeEnable = 0x00100000
const Control0HCTLHSEnable = 0x00000004
const Control0HCTLDataWidth = 0x00000002

// control register 1
const C1ResetData = 0x04000000
const C1ResetCommand = 0x02000000
const C1ResetHost = 0x01000000
const C1TOUNIT_DIS = 0x000f0000
const C1_TOUNIT_MAX = 0x000e0000
const C1ClockGenerationSelect = 0x00000020
const C1ClockEnable = 0x00000004
const C1ClockStable = 0x00000002
const C1ClockEnableInternal = 0x00000001

// STATUS register settings
const SRReadAvailable = 0x00000800
const SRDataInhibit = 0x00000002
const SRCommandInhibit = 0x00000001
const SRAppCommand = 0x00000020

// COMMANDs
const CommandGoIdle = 0x00000000
const CommandAllSendCID = 0x02010000
const CommandSendRelAddr = 0x03020000
const CommandCardSelect = 0x07030000
const CommandSendIfCond = 0x08020000
const CommandStopTrans = 0x0C030000
const CommandReadSingle = 0x11220010
const CommandReadMulti = 0x12220032
const CommandSetBlockcount = 0x17020000
const CommandAppCommand = 0x37000000
const CommandSetBusWidth = (0x06020000 | CommandNeedApp)
const CommandSendOpCond = (0x29020000 | CommandNeedApp)
const CommandSendSCR = (0x33220010 | CommandNeedApp)

// command flags
const CommandNeedApp = 0x80000000
const CommandResponse48 = 0x00020000
const CommandErrorsMask = 0xfff9c004
const CommandRCAMask = 0xffff0000

// status
const SDOk = 0
const SDTimeout = -1
const SDError = -2
const SDTimeoutUnsigned = 0xffffffffffffffff
const SDErrorUnsigned = 0xfffffffffffffffe

// INTERRUPT register settings
const InterruptDataTimeout = 0x00100000
const InterruptCommandTimeout = 0x00010000
const InterruptReadReady = 0x00000020
const InterruptCommandDone = 0x00000001

const InterruptErrorMask = 0x017E8000

const ACMD41_VOLTAGE = 0x00ff8000
const ACMD41_CMD_COMPLETE = 0x80000000
const ACMD41_CMD_CCS = 0x40000000
const ACMD41_ARG_HC = 0x51ff8000

// SCR flags
const SCR_SD_BUS_WIDTH_4 = 0x00000400
const SCR_SUPP_SET_BLKCNT = 0x02000000

// added by bzt driver
const SCR_SUPP_CCS = 0x00000001

/*

#define EMMC_ARG2           ((volatile unsigned int*)(MMIO_BASE+0x00300000))
#define EMMC_BLKSIZECNT     ((volatile unsigned int*)(MMIO_BASE+0x00300004))
#define EMMC_ARG1           ((volatile unsigned int*)(MMIO_BASE+0x00300008))
#define EMMC_CMDTM          ((volatile unsigned int*)(MMIO_BASE+0x0030000C))
#define EMMC_RESP0          ((volatile unsigned int*)(MMIO_BASE+0x00300010))
#define EMMC_RESP1          ((volatile unsigned int*)(MMIO_BASE+0x00300014))
#define EMMC_RESP2          ((volatile unsigned int*)(MMIO_BASE+0x00300018))
#define EMMC_RESP3          ((volatile unsigned int*)(MMIO_BASE+0x0030001C))
#define EMMC_DATA           ((volatile unsigned int*)(MMIO_BASE+0x00300020))
#define EMMC_STATUS         ((volatile unsigned int*)(MMIO_BASE+0x00300024))
#define EMMC_CONTROL0       ((volatile unsigned int*)(MMIO_BASE+0x00300028))
#define EMMC_CONTROL1       ((volatile unsigned int*)(MMIO_BASE+0x0030002C))
#define EMMC_INTERRUPT      ((volatile unsigned int*)(MMIO_BASE+0x00300030))
#define EMMC_INT_MASK       ((volatile unsigned int*)(MMIO_BASE+0x00300034))
#define EMMC_INT_EN         ((volatile unsigned int*)(MMIO_BASE+0x00300038))
#define EMMC_CONTROL2       ((volatile unsigned int*)(MMIO_BASE+0x0030003C))
#define EMMC_SLOTISR_VER    ((volatile unsigned int*)(MMIO_BASE+0x003000FC))

// command flags
#define CMD_NEED_APP        0x80000000
#define CMD_RSPNS_48        0x00020000
#define CMD_ERRORS_MASK     0xfff9c004
#define CMD_RCA_MASK        0xffff0000

// COMMANDs
#define CMD_GO_IDLE         0x00000000
#define CMD_ALL_SEND_CID    0x02010000
#define CMD_SEND_REL_ADDR   0x03020000
#define CMD_CARD_SELECT     0x07030000
#define CMD_SEND_IF_COND    0x08020000
#define CMD_STOP_TRANS      0x0C030000
#define CMD_READ_SINGLE     0x11220010
#define CMD_READ_MULTI      0x12220032
#define CMD_SET_BLOCKCNT    0x17020000
#define CMD_APP_CMD         0x37000000
#define CMD_SET_BUS_WIDTH   (0x06020000|CMD_NEED_APP)
#define CMD_SEND_OP_COND    (0x29020000|CMD_NEED_APP)
#define CMD_SEND_SCR        (0x33220010|CMD_NEED_APP)

// STATUS register settings
#define SR_READ_AVAILABLE   0x00000800
#define SR_DAT_INHIBIT      0x00000002
#define SR_CMD_INHIBIT      0x00000001
#define SR_APP_CMD          0x00000020

// INTERRUPT register settings
#define INT_DATA_TIMEOUT    0x00100000
#define INT_CMD_TIMEOUT     0x00010000
#define INT_READ_RDY        0x00000020
#define INT_CMD_DONE        0x00000001

#define INT_ERROR_MASK      0x017E8000

// CONTROL register settings
#define C0_SPI_MODE_EN      0x00100000
#define C0_HCTL_HS_EN       0x00000004
#define C0_HCTL_DWITDH      0x00000002

#define C1_SRST_DATA        0x04000000
#define C1_SRST_CMD         0x02000000
#define C1_SRST_HC          0x01000000
#define C1_TOUNIT_DIS       0x000f0000
#define C1_TOUNIT_MAX       0x000e0000
#define C1_CLK_GENSEL       0x00000020
#define C1_CLK_EN           0x00000004
#define C1_CLK_STABLE       0x00000002
#define C1_CLK_INTLEN       0x00000001

// SLOTISR_VER values
#define HOST_SPEC_NUM       0x00ff0000
#define HOST_SPEC_NUM_SHIFT 16
#define HOST_SPEC_V3        2
#define HOST_SPEC_V2        1
#define HOST_SPEC_V1        0

// SCR flags
#define SCR_SD_BUS_WIDTH_4  0x00000400
#define SCR_SUPP_SET_BLKCNT 0x02000000
// added by my driver
#define SCR_SUPP_CCS        0x00000001

#define ACMD41_VOLTAGE      0x00ff8000
#define ACMD41_CMD_COMPLETE 0x80000000
#define ACMD41_CMD_CCS      0x40000000
#define ACMD41_ARG_HC       0x51ff8000

unsigned long sd_scr[2], sd_ocr, sd_rca, sd_err, sd_hv;
*/
