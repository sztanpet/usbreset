// Reset USB device, point is to not have to physically re-plug the device
// Reimplementation of code from https://askubuntu.com/questions/645/how-do-you-reset-a-usb-device-from-the-command-line
// Use lsusb to identify vendor/product on the bus.
// Requires linux and devfs to be mounted on /dev
//
// Example: ./usbreset -v 1a86 -p 7523 -d true
// lsusb output was:
// Bus 003 Device 002: ID 1a86:7523 QinHeng Electronics CH340 serial converter

package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/sys/unix"
)

var vendor = flag.String("vendor", "", "The USB vendor id of the usb device that shall be reset (ex: 1a86")
var product = flag.String("product", "", "The USB product id of the usb device that shall be reset (ex: 7523)")
var resetPath = flag.String("resetPath", "", "The USB device path to reset")
var debug = flag.Bool("debug", false, "debug mode")

func init() {
	flag.StringVar(vendor, "v", "", "(shorthand for --vendor)")
	flag.StringVar(product, "p", "", "(shorthand for --product)")
	flag.StringVar(resetPath, "rp", "", "(shorthand for --resetPath)")
	flag.BoolVar(debug, "d", false, "(shorthand for --debug)")
}

func getPathFromVendorProduct() string {
	var resetPath string
	// look at usb_device types only (usb_interface devices have a directory
	// in the form of 1-1:1-1), we want to reset devices
	m, err := filepath.Glob("/sys/bus/usb/devices/[0-9]*-[0-9]*/uevent")
	if err != nil {
		fail("Could not glob /sys/bus/usb/devices/", err)
	}

	// the uevent file will contain DEVTYPE=usb_device, we assume that is the case
	// it will also contain PRODUCT=1a86/7523/264 this is what we look for in the
	// file, and if found, we are interested in the BUSNUM= and DEVNUM= lines
	search := []byte("PRODUCT=" + *vendor + "/" + *product)
	for _, p := range m {
		c, err := ioutil.ReadFile(p)
		if err != nil {
			fail("Could not read file: "+p, err)
		}

		if !bytes.Contains(c, search) {
			d("Did not find " + string(search) + " in\t" + p)
			continue
		}

		d("Found " + string(search) + " in\t\t" + p)
		lines := bytes.Split(c, []byte("\n"))
		var bus string
		var dev string
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("BUSNUM=")) {
				bus = string(bytes.TrimPrefix(line, []byte("BUSNUM=")))
			}
			if bytes.HasPrefix(line, []byte("DEVNUM=")) {
				dev = string(bytes.TrimPrefix(line, []byte("DEVNUM=")))
			}
		}

		if bus == "" || dev == "" {
			fail("Could not find bus/dev, contents were: "+string(c), nil)
		}

		// the device filesystem could be mounted anywhere, not just /dev
		// but we assume its mounted on /dev
		resetPath = path.Join("/dev/bus/usb", bus, dev)
		break
	}

	return resetPath
}
func main() {
	flag.Parse()

	if *resetPath == "" && (*vendor == "" || *product == "") {
		flag.PrintDefaults()
		fail("No path/vendor/product to reset specified!", nil)
	}

	if *resetPath == "" {
		*resetPath = getPathFromVendorProduct()
	}

	if *resetPath == "" {
		fail("Could not ascertain path to reset", nil)
	}

	d("Resetting: " + *resetPath)
	h, err := os.OpenFile(*resetPath, os.O_WRONLY, 666)
	if err != nil {
		fail("Could not open file", err)
	}

	// send the magic ioctl incantation to reset the device
	const USBDEVFS_RESET = 'U'<<(4*2) | 20
	if err = unix.IoctlSetInt(int(h.Fd()), USBDEVFS_RESET, 0); err != nil {
		fail("Could not reset", err)
	}

	_ = h.Close()
	os.Exit(0)
}

func fail(msg string, err error) {
	fmt.Print(msg)
	if err != nil {
		fmt.Println(": " + err.Error())
	} else {
		fmt.Println()
	}

	os.Exit(1)
}

func d(msg string) {
	if *debug {
		fmt.Println(msg)
	}
}
