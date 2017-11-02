package otb

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
)

// OTB reads in the file format as implemented in OpenTibia Server's fileloader.cpp.
type OTB struct {
	reader io.ReadSeeker
}

const (
	ESCAPE_CHAR = 0xFD
	NODE_START  = 0xFE
	NODE_END    = 0xFF
)

func NewOTB(r io.ReadSeeker) (*OTB, error) {
	otb := OTB{
		reader: r,
	}

	var version uint32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("error reading otb version: %v", err)
	}
	if version > 0 {
		return nil, fmt.Errorf("invalid otb version; got %d, want %d", version, 0)
	}

	var byt uint8
	if err := binary.Read(r, binary.LittleEndian, &byt); err != nil {
		return nil, fmt.Errorf("error starting reading otb node: %v", err)
	}
	if byt == NODE_START {
		otb.parseNode()
	} else {
		return nil, fmt.Errorf("bad otb: expected start of node: got %x, want %x", byt, NODE_START)
	}

	return &otb, nil
}

type OTBNode struct {
	nodeType uint8
	child    *OTBNode
	next     *OTBNode
}

func (otb *OTB) parseNode() (*OTBNode, error) {
	node := OTBNode{}
	if err := node.parse(otb.reader, 0); err != nil {
		return nil, err
	} else {
		return &node, nil
	}
}

func (n *OTBNode) parse(reader io.ReadSeeker, depth int) error {
	currentNode := n
	for {
		var nodeType uint8
		if err := binary.Read(reader, binary.LittleEndian, &nodeType); err != nil {
			return fmt.Errorf("error reading otb node type: %v", err)
		}
		glog.Infof("%stype 0x%02X", strings.Repeat(" ", depth), nodeType)
		currentNode.nodeType = nodeType

		var props []uint8
		for {
			shouldBreakFor := false

			var byt uint8
			if err := binary.Read(reader, binary.LittleEndian, &byt); err != nil {
				return fmt.Errorf("error reading otb byte: %v", err)
			}
			switch byt {
			case NODE_START:
				props = []uint8{}
				node := OTBNode{}
				currentNode.child = &node
				if err := node.parse(reader, depth+1); err != nil {
					return fmt.Errorf("error parsing child node: %v", err)
				}
			case NODE_END:
				var byt uint8
				if err := binary.Read(reader, binary.LittleEndian, &byt); err != nil {
					return fmt.Errorf("error reading otb byte: %v", err)
				}
				switch byt {
				case NODE_START:
					node := OTBNode{}
					currentNode.next = &node
					currentNode = &node
					shouldBreakFor = true
					// TODO(ivucica): why not just parse the subnode here?
				case NODE_END:
					glog.Infof("props: %+v", props)
					return nil
				default:
					return fmt.Errorf("expected NODE_START or NODE_END, got %x", byt)
				}
			case ESCAPE_CHAR:
				// Skip one byte. TODO(ivucica): simplify... too lazy to look up what's offered by io.ReadSeeker to skip 1 byte
				var byt uint8
				if err := binary.Read(reader, binary.LittleEndian, &byt); err != nil {
					return fmt.Errorf("error reading otb byte: %v", err)
				}
				byt = byt

				// TEMP construction of props
				props = append(props, byt)
			default:
				// TEMP construction of props
				props = append(props, byt)
			}

			if shouldBreakFor {
				glog.Infof("props: %+v", props)
				props = []uint8{}
				break
			}
		}
	}
	return nil
}
