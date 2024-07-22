package gwmap

import (
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/things"
)

type Creature interface {
	GetPos() tnet.Position
	SetPos(tnet.Position) error
	GetID() CreatureID
	GetName() string
	GetDir() things.CreatureDirection // TODO: move to tnet? or move tnet.Position to things?
	SetDir(things.CreatureDirection) error
	GetServerType() uint16
	GetOutfitColors() [4]things.OutfitColor
}

type CreatureID uint32
