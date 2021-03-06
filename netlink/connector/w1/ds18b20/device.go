package ds18b20

import (
	"github.com/SpComb/go-onewire/netlink/connector/w1"
)

func MakeDevice(conn *w1.Conn, id w1.SlaveID) Device {
	return Device{
		conn: conn,
		id:   id,
	}
}

type Device struct {
	conn *w1.Conn
	id   w1.SlaveID
}

func (d *Device) String() string {
	return d.id.String()
}

func (d *Device) Cmd(cmd Cmd, write []byte, read []byte) error {
	write = append([]byte{byte(cmd)}, write...)

	return d.conn.CmdSlave(d.id, write, read)
}

// XXX: need to wait 100..800ms for conversion to happen?
func (d *Device) ConvertT() error {
	return d.Cmd(CmdConvertT, nil, nil)
}

// Checks CRC, fails on bus errors
func (d *Device) Read() (Scratchpad, error) {
	var scratchpad Scratchpad
	var read = make([]byte, scratchpadSize)

	if err := d.Cmd(CmdReadScratchpad, nil, read); err != nil {
		return scratchpad, err
	}

	if err := scratchpad.unpack(read); err != nil {
		return scratchpad, err
	}

	return scratchpad, nil
}

// NOTE: this does not check CRCs, and will return an invalid temperature on bus errors
func (d *Device) ReadTemperature() (Temperature, error) {
	var read = make([]byte, 2)

	if err := d.Cmd(CmdReadScratchpad, nil, read); err != nil {
		return 0, err
	}

	return unpackTemperature(read[0], read[1]), nil
}
