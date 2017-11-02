package otb

import (
	"fmt"
	"io"

	"github.com/golang/glog"
)

type ItemsOTB struct {
	OTB
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
	if otb.ChildNode(root) == nil {
		return nil, fmt.Errorf("no children in root node")
	}
	for node := otb.ChildNode(root); node != nil; node = node.NextNode() {
		glog.Infof("type: %02x", node.NodeType())
	}

	return &otb, nil
}
