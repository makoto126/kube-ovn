package jmnd

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
	"libvirt.org/go/libvirt"
)

const (
	backendVM = "kubeovnVM"

	libvirtURL = "jmnd+tcp://127.0.0.1/system"

	initxml = `
<domain type='kvm' xmlns:qemu='http://libvirt.org/schemas/domain/qemu/1.0'>
  <name>%s</name>
  <uuid>%s</uuid>
  <devices>
  </devices>
  <qemu:commandline>
    <qemu:arg value='-jmnd'/>
    <qemu:arg value='vm-type=8,boot-mode=0,action-type=0'/>
  </qemu:commandline>
</domain>	
`

	netxml = `
<interface type='vhostuser'>
  <mac address='%s'/>
  <model type='virtio-transitional'/>
  <driver queues='4' mq='on'/>
  <source type='unix' path='/tmp/sock_%s#bdf=%s' mode='server'/>
</interface>
`
)

var (
	libvirtCli *libvirt.Connect
	csivm      *libvirt.Domain
)

func LibvirtInit() error {
	var err error

	libvirtCli, err = libvirt.NewConnect(libvirtURL)
	if err != nil {
		klog.Errorf("Failed to connect libvirt: %v", err)
		return err
	}
	csivm, err = libvirtCli.LookupDomainByName(backendVM)
	if err == nil {
		//already created
		return nil
	}
	csivm, err = libvirtCli.DomainCreateXML(
		fmt.Sprintf(initxml, backendVM, uuid.New().String()),
		libvirt.DOMAIN_NONE,
	)
	if err != nil {
		klog.Errorf("Failed to create backendVM: %v", err)
		return err
	}

	klog.Infof("backendVM created")

	return nil
}

func LibvirtAttachDev(macAddr, portName, bdf string) error {
	return csivm.AttachDevice(fmt.Sprintf(netxml, macAddr, portName, strings.TrimPrefix(bdf, "0000:")))
}

func LibvirtDetachDev(macAddr, portName, bdf string) error {
	return csivm.DetachDevice(fmt.Sprintf(netxml, macAddr, portName, strings.TrimPrefix(bdf, "0000:")))
}
