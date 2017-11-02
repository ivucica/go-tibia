package otb

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang/glog"
)

type ItemsOTB struct {
	OTB
}

type ItemsOTBAttribute uint8
type ItemsOTBDataSize uint16

const (
	ROOT_ATTR_VERSION = 0x01
)

type ItemsOTBRootNodeVersion struct {
	DataSize ItemsOTBDataSize
	Version  struct {
		MajorVersion, MinorVersion, BuildNumber uint32
		CSDVersion                              [128]uint8
	}
}

func NewItemsOTB(r io.ReadSeeker) (*ItemsOTB, error) {
	f, err := NewOTB(r)
	if err != nil {
		return nil, fmt.Errorf("newitemsotb failed to use fileloader: %s", err)
	}

	otb := ItemsOTB{
		OTB: *f,
	}

	root := otb.ChildNode(nil)
	if root == nil {
		return nil, fmt.Errorf("nil root node")
	}

	props := root.PropsBuffer()
	var flags uint32
	if err := binary.Read(props, binary.LittleEndian, &flags); err != nil {
		return nil, fmt.Errorf("error reading itemsotb root node flags: %v", err)
	}
	flags = flags // seemingly unused

	var attr ItemsOTBAttribute
	if err := binary.Read(props, binary.LittleEndian, &attr); err != nil {
		return nil, fmt.Errorf("error reading itemsotb root node attr: %v", err)
	}
	switch attr {
	case ROOT_ATTR_VERSION:
		var vers ItemsOTBRootNodeVersion
		if err := binary.Read(props, binary.LittleEndian, &vers); err != nil {
			return nil, fmt.Errorf("error reading itemsotb root node attr 'version': %v", err)
		}
		if vers.DataSize != /* sizeof ItemsOTBRootNodeVersion */ 4+4+4+128 {
			return nil, fmt.Errorf("bad size of itemsotb root node attr 'version': %v", vers.DataSize)
		}
	default:
		// ignore, apparently
	}

	if otb.ChildNode(root) == nil {
		return nil, fmt.Errorf("no children in root node")
	}
	for node := otb.ChildNode(root); node != nil; node = node.NextNode() {
		glog.Infof("type: %02x", node.NodeType())
	}

	return &otb, nil
}
