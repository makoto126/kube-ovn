package cni

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	Sriov           = "jmnd_sriov"
	Virtio          = "virtio-pci"
	Vfio            = "vfio-pci"
	DriverOverride  = "/sys/bus/pci/devices/%s/driver_override"
	DriverUnbind    = "/sys/bus/pci/devices/%s/driver/unbind"
	DriverProbe     = "/sys/bus/pci/drivers_probe"
	VirtioInterface = "/sys/bus/pci/devices/%s/virtio*/net/*"
	VirtioDriver    = "/sys/bus/pci/drivers/virtio-pci/%s"
)

func statWrite(fname string, data []byte) error {
	fi, err := os.Stat(fname)
	if err != nil {
		return err
	}
	return os.WriteFile(fname, data, fi.Mode())
}

func override(driver string, addr string) error {
	fname := fmt.Sprintf(DriverOverride, addr)

	return statWrite(fname, []byte(driver))
}

func unbind(addr string) error {
	fname := fmt.Sprintf(DriverUnbind, addr)

	return statWrite(fname, []byte(addr))
}

func probe(addr string) error {
	return statWrite(DriverProbe, []byte(addr))
}

func SetDriver(deviceID, drivename string) error {
	var err error

	err = override(drivename, deviceID)
	if err != nil {
		return err
	}

	_ = unbind(deviceID)

	err = probe(deviceID)

	return err
}

func IsVirtioDevice(deviceID string) bool {
	fi, _ := os.Stat(fmt.Sprintf(VirtioDriver, deviceID))
	return fi != nil
}

func getVirtioInterfaceName(pciAddr string) string {
	name := ""
	retry_cnt := 5
	pattern := fmt.Sprintf(VirtioInterface, pciAddr)

	for retry_cnt > 0 {
		dirs, _ := filepath.Glob(pattern)
		if len(dirs) > 0 {
			name = filepath.Base(dirs[0])
			if name != "" && name != "eth0" {
				return name
			}
		}
		time.Sleep(1 * time.Second)
		retry_cnt--
	}

	return ""
}
