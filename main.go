// Reset USB device, point is to not have to physically re-plug the device
// Reimplementation of code from https://askubuntu.com/questions/645/how-do-you-reset-a-usb-device-from-the-command-line
// Use lsusb to identify vendor/product on the bus
// Example: ./usbreset -v 1a86 -p 7523 -d true

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
var product = flag.String("product", "", "The USB product id of the usb device that shall be reset (ex: 7523")
var debug = flag.Bool("debug", false, "debug mode")

func init() {
	flag.StringVar(vendor, "v", "", "(shorthand for --vendor)")
	flag.StringVar(product, "p", "", "(shorthand for --product)")
	flag.BoolVar(debug, "d", false, "(shorthand for --debug)")
}

func main() {
	flag.Parse()

	if *vendor == "" || *product == "" {
		flag.PrintDefaults()
		fail("No vendor/product to reset specified!", nil)
	}
	var pathToReset string

	m, err := filepath.Glob("/sys/bus/usb/devices/[0-9]-[0-9]/uevent")
	if err != nil {
		fail("Could not glob /sys/bus/usb/devices/", err)
	}

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

			continue
		}

		if bus == "" || dev == "" {
			fail("Could not find bus/dev, contents were: "+string(c), nil)
		}

		pathToReset = path.Join("/dev/bus/usb", bus, dev)
		break
	}

	if pathToReset == "" {
		fail("Could not ascertain path to reset", nil)
	}

	d("Resetting: " + pathToReset)
	h, err := os.OpenFile(pathToReset, os.O_WRONLY, 666)
	if err != nil {
		fail("Could not open file", err)
	}

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
