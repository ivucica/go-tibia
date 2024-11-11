package otbm

import (
	"encoding/binary"
	"fmt"
	"io"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/otb"
	"badc0de.net/pkg/go-tibia/things"

	"github.com/golang/glog"
)

// New reads an OTB file from a given reader.
func New(r io.ReadSeeker, t *things.Things) (*Map, error) {
	f, err := otb.NewOTB(r)
	if err != nil {
		return nil, fmt.Errorf("newotbm failed to use fileloader: %s", err)
	}

	otb := Map{
		tiles:     map[pos]*mapTile{},
		creatures: map[gameworld.CreatureID]gameworld.Creature{},

		things: t,
	}

	root := f.ChildNode(nil)
	if root == nil {
		return nil, fmt.Errorf("nil root node")
	}

	props := root.PropsBuffer()
	//var attr MapAttribute
	//if err := binary.Read(props, binary.LittleEndian, &attr); err != nil {
	//	return fmt.Errorf("error reading otbm root node attr: %v", err)
	//}
	switch MapNodeType(root.NodeType()) {
	case OTBM_ROOT:
		var head rootHeader
		if err := binary.Read(props, binary.LittleEndian, &head); err != nil {
			return nil, fmt.Errorf("error reading otbm root node header attrs: %v", err)
		}

		glog.V(2).Infof("otbm header: %+v", head)
		// TODO: store version and ensure items.otb is applicable enough
	case OTBM_ROOTV1:
		return nil, fmt.Errorf("otbm with rootv1 header is not supported at this time")
	default:
		glog.Errorf("unknown root node 0x%02x", root.NodeType())
		return nil, fmt.Errorf("unknown root node 0x%02x", root.NodeType())
	}

	if root.ChildNode() == nil {
		return nil, fmt.Errorf("no children in root node")
	}

	for node := root.ChildNode(); node != nil; node = node.NextNode() {
		if err := otb.readRootChildNode(node); err != nil {
			return nil, fmt.Errorf("error reading root child node: %v", err)
		}
	}

	if otb.defaultPlayerSpawnPoint == 0 {
		//otb.defaultPlayerSpawnPoint = posFromCoord(44, 173, 5) // generated file
		//otb.defaultPlayerSpawnPoint = posFromCoord(1001, 1010, 7) // test file
		//return fmt.Errorf("no default player spawn point; does the map have any temples?")
		glog.Warningf("no default player spawn point; does the map have any temples?")
	}

	return &otb, nil
}
