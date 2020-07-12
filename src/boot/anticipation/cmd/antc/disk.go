package main

import (
	"drivers/emmc"
	"lib/trust"
)

const maddie = "/feelings/madeleine"

var fp emmc.EmmcFile

func canBootFromDisk(logger *trust.Logger) bool {
	var err error
	if emmc.Impl.Init() != emmc.EmmcOk {
		logger.Errorf("Unable to initialize EMMC driver, ",
			"booting from serial port...")
	}
	fp, err = emmc.Impl.Open(maddie)
	if err != nil {
		logger.Errorf("Unable to find %s binary, ",
			"booting from serial port...", maddie)

	}
	logger.Infof("found bootable lady: %s", maddie)
	return true
}

func bootDisk(logger *trust.Logger) {

}
