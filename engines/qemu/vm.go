package qemuengine

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/taskcluster/slugid-go/slugid"
	"github.com/taskcluster/taskcluster-worker/engines/qemu/image"
	"github.com/taskcluster/taskcluster-worker/engines/qemu/network"
)

// virtualMachine holds the QEMU process and associated resources.
// This is useful as the VM remains alive in the ResultSet stage, as we use
// guest tools to copy files from the virtual machine.
type virtualMachine struct {
	m         sync.Mutex // Protect access to resources
	started   bool
	network   *network.Network
	image     *image.Instance
	vncSocket string
	qmpSocket string
	qemu      *exec.Cmd
	qemuDone  chan<- struct{}
	Done      <-chan struct{} // Closed when the virtual machine is done
	Error     error           // Error, to be read after Done is closed
}

// newVirtualMachine constructs a new virtual machine.
func newVirtualMachine(
	image *image.Instance, network *network.Network, socketFolder string,
) *virtualMachine {
	// Construct virtual machine
	vm := &virtualMachine{
		vncSocket: filepath.Join(socketFolder, slugid.V4()+".sock"),
		qmpSocket: filepath.Join(socketFolder, slugid.V4()+".sock"),
		network:   network,
		image:     image,
	}

	// Construct options for QEMU
	type opts map[string]string
	arg := func(kind string, opts opts) string {
		result := kind
		for k, v := range opts {
			if result != "" {
				result += ","
			}
			result += k + "=" + v
		}
		return result
	}
	options := []string{
		"-name", "qemu-guest",
		// TODO: Add -enable-kvm (configurable so can be disabled in tests)
		"-machine", "pc-i440fx-2.1", // TODO: Configure additional options
		"-m", "512", // TODO: Make memory configurable
		"-realtime", "mlock=off", // TODO: Enable for things like talos
		// TODO: fit to system HT, see: https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-devices-system-cpu
		// TODO: Configure CPU instruction sets: http://forum.ipfire.org/viewtopic.php?t=12642
		"-smp", "cpus=2,sockets=2,cores=1,threads=1",
		"-uuid", "9372acf5-d413-4572-bc7b-7a1d2df57bab", // TODO: allow customization
		"-no-user-config", "-nodefaults",
		"-rtc", "base=utc", // TODO: Allow clock=vm for loadvm with windows
		"-boot", "menu=off,strict=on",
		"-device", arg("VGA", opts{ // TODO: Investigate if we can use vmware
			"id":        "video-0",
			"vgamem_mb": "64", // TODO: Customize VGA memory
			"bus":       "pci.0",
			"addr":      "0x2", // QEMU uses PCI 0x2 for VGA by default
		}),
		"-device", arg("nec-usb-xhci", opts{
			"id":   "usb",
			"bus":  "pci.0",
			"addr": "0x3", // Always put USB on PCI 0x3
		}),
		"-device", arg("virtio-balloon-pci", opts{
			"id":   "balloon-0",
			"bus":  "pci.0",
			"addr": "0x4", // Always put balloon on PCI 0x4
		}),
		"-netdev", vm.network.NetDev("netdev-0"),
		"-device", arg(vm.image.Machine().Network.Device, opts{
			"netdev": "netdev-0",
			"id":     "nic0",
			"mac":    vm.image.Machine().Network.MAC,
			"bus":    "pci.0",
			"addr":   "0x5", // Always put network on PCI 0x5
		}),
		"-device", arg("AC97", opts{ // TODO: Customize sound device
			"id":   "sound-0",
			"bus":  "pci.0",
			"addr": "0x6", // Always put sound on PCI 0x6
		}),
		"-device", arg("usb-kbd", opts{
			"id":   "keyboard-0",
			"bus":  "usb.0",
			"port": "0",
		}),
		"-device", arg("usb-mouse", opts{
			"id":   "mouse-0",
			"bus":  "usb.0",
			"port": "1",
		}),
		"-vnc", arg("unix:"+vm.vncSocket, opts{
			"share": "force-shared",
		}),
		"-chardev", "socket,id=qmp-socket,path=" + vm.qmpSocket + ",nowait,server=on",
		"-qmp", "qmp-socket",
		"-drive", arg("", opts{
			"file":   vm.image.DiskFile(),
			"if":     "none",
			"id":     "boot-disk",
			"cache":  "unsafe",
			"aio":    "native",
			"format": "qcow2",
			"werror": "report",
			"rerror": "report",
		}),
		"-device", arg("virtio-blk-pci", opts{
			"scsi":      "off",
			"bus":       "pci.0",
			"addr":      "0x8", // Start disks as 0x8, reserve 0x7 for future
			"drive":     "boot-disk",
			"id":        "virtio-disk0",
			"bootindex": "1",
		}),
		// TODO: Add cache volumes
	}

	// Create done channel
	qemuDone := make(chan struct{})
	vm.qemuDone = qemuDone
	vm.Done = qemuDone

	// Create QEMU process
	vm.qemu = exec.Command("qemu-system-x86_64", options...)

	return vm
}

func (vm *virtualMachine) SetHTTPHandler(handler http.Handler) {
	vm.m.Lock()
	defer vm.m.Unlock()
	if vm.network != nil {
		// Ignore the case where network has been released
		vm.network.SetHandler(handler)
	}
}

// Start the virtual machine.
func (vm *virtualMachine) Start() {
	vm.m.Lock()
	if vm.started {
		vm.m.Unlock()
		panic("virtual machine instance have already been started once")
	}
	vm.started = true
	vm.m.Unlock()

	// Start QEMU
	vm.Error = vm.qemu.Start()
	if vm.Error != nil {
		close(vm.qemuDone)
		return
	}

	// Wait for QEMU to finish before closing Done
	go func(vm *virtualMachine) {
		vm.Error = vm.qemu.Wait()

		// Release network and image
		vm.m.Lock()
		defer vm.m.Unlock()
		vm.network.Release()
		vm.network = nil
		vm.image.Release()
		vm.image = nil

		// Remove socket files
		os.Remove(vm.vncSocket)
		os.Remove(vm.qmpSocket)
		vm.vncSocket = ""
		vm.qmpSocket = ""

		// Notify everybody that the VM is stooped
		// Ensure resources are freed first, otherwise we'll race with resources
		// against the next task. If the number of resources is limiting the
		// number of concurrent tasks we can run.
		// This is usually the case, so race would happen at full capacity.
		close(vm.qemuDone)
	}(vm)
}

// Kill the virtual machine, can only be called after Start()
func (vm *virtualMachine) Kill() {
	select {
	case <-vm.Done:
		return // We're obviously not running, so we must be done
	default:
		vm.qemu.Process.Kill()
	}
}

// VNCSocket returns the path to VNC socket, empty-string if closed.
func (vm *virtualMachine) VNCSocket() string {
	// Lock access to vncSocket
	vm.m.Lock()
	defer vm.m.Unlock()

	return vm.vncSocket
}
