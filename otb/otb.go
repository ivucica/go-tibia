package otb

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/golang/glog"
)

// OTB reads in the file format as implemented in OpenTibia Server's fileloader.cpp.
//
// The implementation currently vaguely mirrors what's in fileloader.cpp. It's
// somewhat suboptimal in the way it processes children, stores 'props' (non-node
// blobs attached to a node) and such. It avoids some deficiencies in the
// reference implementation, but is very suboptimal when it comes to parsing this
// file format.
//
// It could be supplanted by a smarter, more Go-like file format reader.
type OTB struct {
	reader io.ReadSeeker

	root *OTBNode
}

// Various special-meaning characters that might be encountered while parsing a
// node.
const (
	ESCAPE_CHAR = 0xFD // Following character should be read verbatim, even if it otherwise has a special meaning.
	NODE_START  = 0xFE // From this character onwards, this is a new OTB node. If preceded by NODE_END, this is the next sibling node. Otherwise, it's a child node.
	NODE_END    = 0xFF // This character marks the end of the latest OTB node. If immediately followed by a NODE_START, that will be the next sibling node.
)

// NewOTB reads an OTB file from the given `io.ReadSeeker`, and constructs a
// tree of nodes.
//
// No meaning is assigned to nodes; this is the task of readers for an
// individual format.
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
		root, err := otb.parseNode()
		if err != nil {
			return nil, fmt.Errorf("bad otb: could not parse root node: %s", err)
		}
		otb.root = root
	} else {
		return nil, fmt.Errorf("bad otb: expected start of node: got %x, want %x", byt, NODE_START)
	}

	return &otb, nil
}

// parseNode processes the next node in the given reader.
func (otb *OTB) parseNode() (*OTBNode, error) {
	node := OTBNode{}
	if err := node.parse(otb.reader, 0); err != nil {
		return nil, err
	} else {
		return &node, nil
	}
}

// ChildNode returns whichever is the first child node of a given node. If nil
// is passed, root node is assumed.
//
// To obtain further children, use child's NextNode to obtain the first
// sibling, then use that child's NextNode to obtain the next sibling, etc.
//
// TODO(ivucica): Refactor this. These calls should be made on OTBNode.
func (otb *OTB) ChildNode(parent *OTBNode) *OTBNode {
	if parent == nil {
		return otb.root
	}
	return parent.ChildNode()
}

// OTBNode represents a single node in an OTB-formatted file.
//
// Each node has a type, some data, and may have a sibling and a child attached
// to it.
//
// Further meaning depends on the file; for example, root node in items.otb
// does not use type, but uses data array to store the version of the file and
// a free form descriptor. Its child is the first item in the file; item's
// sibling is the second item; second item's sibling is the third item; et
// cetera.
type OTBNode struct {
	nodeType uint8
	props    []byte
	child    *OTBNode
	next     *OTBNode
}

// NodeType returns the type of the node.
//
// For example, in item nodes in items.otb, this means which item group the
// item belongs to (item groups being used in editors to group items into
// sections such as ground, walls, etc).
func (n *OTBNode) NodeType() uint8 {
	return n.nodeType
}

// ChildNode returns the first child of the node, or null if there are no
// children.
func (n *OTBNode) ChildNode() *OTBNode {
	return n.child
}

// NextNode returns the next sibling of the node, or null if there are no
// more siblings.
func (n *OTBNode) NextNode() *OTBNode {
	return n.next
}

// Returns a new (i.e. reset to start) buffer for reading properties.
func (n *OTBNode) PropsBuffer() *bytes.Buffer {
	return bytes.NewBuffer(n.props)
}

// parse reads all the bytes associated with the current node, as well as its
// children.
//
// It expects that NODE_START byte has already been read.
func (n *OTBNode) parse(reader io.ReadSeeker, depth int) error {
	currentNode := n
	for {
		var nodeType uint8
		if err := binary.Read(reader, binary.LittleEndian, &nodeType); err != nil {
			if err == io.EOF {
				if depth != 0 {
					glog.Warning("warning: abrupt end to an OTB.")
				}
				return nil
			}
			return fmt.Errorf("error reading otb node type: %v", err)
		}
		glog.V(3).Infof("%stype 0x%02X", strings.Repeat(" ", depth), nodeType)
		currentNode.nodeType = nodeType

		for {
			shouldBreakFor := false

			var byt uint8
			if err := binary.Read(reader, binary.LittleEndian, &byt); err != nil {
				if err == io.EOF {
					if depth != 0 {
						glog.Warning("warning: abrupt end to an OTB.")
					}
					return nil
				}
				return fmt.Errorf("error reading otb byte: %v", err)
			}
			switch byt {
			case NODE_START:
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
					// glog.Infof("props: %+v", currentNode.props)
					node := OTBNode{}
					currentNode.next = &node
					currentNode = &node
					shouldBreakFor = true
					// TODO(ivucica): why not just parse the subnode here?
				case NODE_END:
					// glog.Infof("props: %+v", currentNode.props)
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

				currentNode.props = append(currentNode.props, byt)
			default:
				currentNode.props = append(currentNode.props, byt)
			}

			if shouldBreakFor {
				break
			}
		}
	}
	return nil
}
