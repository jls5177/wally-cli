package wallycli

import (
	"fmt"
	"github.com/google/gousb"
	"github.com/marcinbor85/gohex"
	"os"
	"time"
)

// TeensyFlash: Flashes Teensy boards.
// It opens the firmware file at the provided path, checks it's integrity, wait for the keyboard to be in Flash mode, flashes it and reboots the board.
func teensyFlash(firmwarePath string, s *FlashState) error {
	file, err := os.Open(firmwarePath)
	if err != nil {
		return fmt.Errorf("error while opening firmware: %s", err)
	}
	defer file.Close()

	s.total = ergodoxCodeSize

	firmware := gohex.NewMemory()
	err = firmware.ParseIntelHex(file)
	if err != nil {
		return fmt.Errorf("error while parsing firmware: %s", err)
	}

	ctx := gousb.NewContext()
	ctx.Debug(0)
	defer ctx.Close()
	var dev *gousb.Device

	// Loop until a keyboard is ready to flash
	for {
		devs, _ := ctx.OpenDevices(func(desc *gousb.DeviceDesc) bool {
			if desc.Vendor == gousb.ID(halfKayVendorID) && desc.Product == gousb.ID(halfKayProductID) {
				return true
			}
			return false
		})

		defer func() {
			for _, d := range devs {
				d.Close()
			}
		}()

		if len(devs) > 0 {
			dev = devs[0]
			break
		}
		time.Sleep(1 * time.Second)
	}

	// Detach keyboard from the kernel
	dev.SetAutoDetach(true)

	// Claim usb device
	cfg, err := dev.Config(1)
	defer cfg.Close()
	if err != nil {
		return fmt.Errorf("error while claiming the usb interface: %s", err)
	}

	s.step = InProgress

	// Loop on the firmware data and program
	var addr uint32
	for addr = 0; addr < ergodoxCodeSize; addr += ergodoxBlockSize {
		// set a longer timeout when writing the first block
		if addr == 0 {
			dev.ControlTimeout = 5 * time.Second
		} else {
			dev.ControlTimeout = 500 * time.Millisecond
		}
		// Prepare and write a firmware block
		// https://www.pjrc.com/teensy/halfkay_protocol.html
		buf := make([]byte, ergodoxBlockSize+2)
		buf[0] = byte(addr & 255)
		buf[1] = byte((addr >> 8) & 255)
		block := firmware.ToBinary(addr, ergodoxBlockSize, 255)
		for index := range block {
			buf[index+2] = block[index]
		}

		bytes, err := dev.Control(0x21, 9, 0x0200, 0, buf)
		if err != nil {
			return fmt.Errorf("error while sending firmware bytes: %s", err)
		}

		s.sent += bytes
	}

	buf := make([]byte, ergodoxBlockSize+2)
	buf[0] = byte(0xFF)
	buf[1] = byte(0xFF)
	buf[2] = byte(0xFF)
	_, err = dev.Control(0x21, 9, 0x0200, 0, buf)

	if err != nil {
		return fmt.Errorf("error while rebooting device: %s", err)
	}
	s.step = Finished
	return nil
}
