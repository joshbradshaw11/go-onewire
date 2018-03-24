package w1

import (
	"github.com/SpComb/iot-poc/netlink/connector"

	"fmt"
)

func Dial() (*Conn, error) {
	var conn Conn

	if connectorConn, err := connector.Dial(); err != nil {
		return nil, err
	} else {
		conn.connectorConn = connectorConn
	}

	return &conn, nil
}

type Conn struct {
	connectorConn *connector.Conn
}

func (c *Conn) Send(msg Message) error {
	if connectorMsg, err := packMessage(msg); err != nil {
		return err
	} else if err := c.connectorConn.Send(connectorMsg); err != nil {
		return err
	} else {
		log.Debugf("Send: %#v", msg)

		return nil
	}
}

func (c *Conn) Receive() ([]Message, error) {
	if connectorMsgs, err := c.connectorConn.Receive(); err != nil {
		return nil, fmt.Errorf("connector Receive: %v", err)
	} else if msgs, err := unpackMessages(connectorMsgs); err != nil {
		return nil, fmt.Errorf("w1 unpack %#v: %v", connectorMsgs, err)
	} else {
		log.Debugf("Receive: %#v", msgs)

		return msgs, nil
	}
}

func (c *Conn) Execute(msg Message) ([]Message, error) {
	if connectorMsg, err := packMessage(msg); err != nil {
		return nil, fmt.Errorf("w1 pack %#v: %v", msg, err)
	} else if connectorMsgs, err := c.connectorConn.Execute(connectorMsg); err != nil {
		return nil, fmt.Errorf("Execute %#v: %v", msg, err)
	} else if msgs, err := unpackMessages(connectorMsgs); err != nil {
		return msgs, fmt.Errorf("w1 unpack %#v: %v", connectorMsgs, err)
	} else if err := validateMessages(msg, msgs); err != nil {
		return msgs, fmt.Errorf("Execute: %v", err)
	} else {
		log.Debugf("Execute %#v: %#v", msg, msgs)

		return msgs, nil
	}
}

func (c *Conn) ListMasters() (MasterList, error) {
	var msg = Message{
		Header: Header{
			Type: MsgTypeListMasters,
		},
	}

	msgs, err := c.Execute(msg)
	if err != nil {
		return nil, err
	}

	var masterList MasterList

	for _, msg := range msgs {
		if err := masterList.UnmarshalBinary(msg.Data); err != nil {
			return masterList, fmt.Errorf("Unmarshal %T: %v", masterList, err)
		}
	}

	log.Infof("ListMasters: %v", masterList)

	return masterList, nil
}

func (c *Conn) ListSlaves(masterID MasterID) (SlaveList, error) {
	var msg = Message{
		Header: Header{
			Type: MsgTypeMasterCmd,
			ID:   IDMaster{ID: masterID}.Pack(),
		},
	}
	var cmd = Cmd{
		CmdHeader: CmdHeader{
			Cmd: CmdListSlaves,
		},
	}

	if data, err := MarshalCmd(cmd); err != nil {
		return nil, fmt.Errorf("MarshalCmd: %v", err)
	} else {
		msg.Data = data
	}

	if err := c.Send(msg); err != nil {
		return nil, err
	}

	var slaveList SlaveList
	var cmdAcks = 0

	for {
		// SLAVE_LIST results in multiple response messages with increasing ack, until last message with ack == 0
		connectorMsgs, err := c.connectorConn.Receive()
		if err != nil {
			return slaveList, err
		}

		for _, connectorMsg := range connectorMsgs {
			var msg Message

			if err := msg.UnmarshalBinary(connectorMsg.Data); err != nil {
				return nil, err
			}

			log.Debugf("Receive: %#v", msg)

			cmds, err := UnmarshalCmdList(msg.Data)
			if err != nil {
				return slaveList, fmt.Errorf("UnmarshalCmdList: %v", err)
			}

			for _, cmd := range cmds {
				if connectorMsg.Ack == 0 {
					cmdAcks++

				} else if cmd.Cmd == CmdListSlaves {
					if err := slaveList.UnmarshalBinary(cmd.Data); err != nil {
						return slaveList, fmt.Errorf("Unmarshal %T: %v", slaveList, err)
					}
				} else {
					return nil, fmt.Errorf("Unexpected response cmd %v: %#v", cmd.Cmd, msg)
				}
			}
		}

		if cmdAcks >= 1 {
			break
		}
	}

	log.Infof("ListSlaves %v: %v", masterID, slaveList)

	return slaveList, nil
}

func (c *Conn) Close() error {
	return c.connectorConn.Close()
}
