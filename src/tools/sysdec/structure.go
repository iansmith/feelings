package sysdec

type DeviceDef struct {
	Vendor         string
	VendorID       string
	Name           string
	Series         string
	Version        int
	Description    string
	LicenseText    string
	Cpu            CPUDef
	NumCores       int
	MMIOBindings   map[string]int
	Peripheral     map[string]*PeripheralDef
	Package        string // this comes from the user opts
	SourceFilename string // this is the filename used to create all this
	OutTags        string //this comes from the command line option
	Import         string //this comes from command line option
}

type CPUDef struct {
	Name                string
	Description         string
	Revision            string
	LittleEndian        bool
	MMUPresent          bool
	FPUPresent          bool
	DSPPresent          bool
	ICachePresent       bool
	DCachePresent       bool
	DeviceNumInterrupts int
}

type PeripheralDef struct {
	Name             string //if set, will be ignored, it is copied from the key in map
	Version          int
	Description      string
	PrependToName    string
	AppendToName     string
	HeaderStructName string
	GroupName        string
	AddressBlock     AddressBlockDef
	Interrupt        InterruptDef
	MMIOBase         int
	//BaseAddress      int  why would you ever need this?
	//Size             int
	Access                AccessDef
	Register              map[string]*RegisterDef
	RegistersWithReserved []*RegisterDef //computed by the generator
}

type AddressBlockDef struct {
	BaseAddress int
	Size        int
	Usage       string
}

type InterruptDef struct {
	Name        string
	Description string
	Value       int
}

type RegisterDef struct {
	Name          string //if set, will be ignored, it is copied from the key in map
	Description   string
	AddressOffset int
	Size          int
	Access        AccessDef
	ResetValue    int
	ResetMask     int
	Field         map[string]*FieldDef
	IsReserved    bool //computed internally
	Dim           int
	DimIncrement  int
	//these indice names are not crosschecked nor namespaced
	DimIndices map[string]int
}

type FieldDef struct {
	Name              string
	Description       string
	BitRange          BitRangeDef
	Access            AccessDef
	EnumeratedValue   map[string]*EnumeratedValueDef
	RegName           string //created during processing
	CanRead, CanWrite bool   //created during processing
}

type EnumeratedValueDef struct {
	Name        string //don't bother setting,will be copied from the map
	Description string
	Value       int
	Field       *FieldDef //created during processing
}
