package gameworld

import (
	"fmt"
	"testing"

	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
)

func LoadThingsForTest(t *testing.T) *things.Things {
	if th, err := things.New(); err != nil {
		t.Fatalf("failed to create things container: %v", err)
	} else {
		if err := th.AddItemsOTB(LoadOTBForTest(t)); err != nil {
			t.Fatalf("failed to add items.otb to things container: %v", err)
		}
		return th
	}
	return nil
}
func LoadOTBForTest(t *testing.T) *itemsotb.Items {
	f, err := paths.Open("items.otb")
	if err != nil {
		t.Fatalf("failed to open file: %s", err)
	}
	otb, err := itemsotb.New(f)
	if err != nil {
		t.Fatalf("failed to parse otb: %s", err)
	}
	return otb
}

type accessCountingMapDataSource struct {
	mapDataSource
}

func generateAccessCountingMapTile(x, y uint16, z uint8) (MapTile, error) {
	t, err := generateMapTileImpl(x, y, z)
	if err != nil {
		return nil, err
	}
	return &accessCountingTile{
		MapTile: t,
		x:       x,
		y:       y,
		z:       z,
	}, err
}

type accessCountingTile struct {
	MapTile

	x, y        uint16
	z           uint8
	accessCount int
	tst         *testing.T
}

func (t *accessCountingTile) String() string {
	return fmt.Sprintf("<access counting tile @ %d,%d,%d cnt %d>", t.x, t.y, t.z, t.accessCount)
}
func (t *accessCountingTile) GetItem(idx int) (MapItem, error) {
	t.accessCount++
	if t.tst != nil {
		t.tst.Logf("@ %d,%d,%d accessed idx %d", t.x, t.y, t.z, idx)
	}
	return t.MapTile.GetItem(idx)
}

func TestMapDescriptionRange(t *testing.T) {
	playerID := CreatureID(123)
	gws := &GameworldServer{
		things: LoadThingsForTest(t),
	}
	gwConn := &GameworldConnection{}
	gwConn.clientVersion = 854
	gwConn.server = gws
	//gwConn.conn = conn
	//gwConn.key = key
	gwConn.id = GameworldConnectionID(playerID)

	ds := &accessCountingMapDataSource{
		mapDataSource: *(NewMapDataSource().(*mapDataSource)),
	}
	ds.mapTileGenerator = func(x, y uint16, z uint8) (MapTile, error) {
		mt, err := generateAccessCountingMapTile(x, y, z)
		if err != nil {
			return nil, err
		}
		mt.(*accessCountingTile).tst = t
		return mt, err
	}
	gws.SetMapDataSource(ds)

	ds.AddCreature(&creature{
		id: 123,
		pos: tnet.Position{
			X:     100,
			Y:     100,
			Floor: 7,
		},
		dir:  things.CreatureDirectionSouth,
		look: 128,
		col:  [4]things.OutfitColor{0x0a, 0x0a, 0x0a, 0x0a},
	})

	msg := tnet.NewMessage()
	err := gwConn.initialAppearMap(msg)
	if err != nil {
		t.Errorf("failed to send map: %v", err)
	}
	type oneTC struct {
		x, y  uint16
		floor uint8
		want  int
	}
	tcs := []struct {
		grouping string
		tests    []oneTC
	}{
		{
			grouping: "top left edge",
			tests: []oneTC{
				{
					x: 91, y: 94, floor: 7,
					want: 0,
				},
				{
					x: 92, y: 93, floor: 7,
					want: 0,
				},
				{
					x: 92, y: 94, floor: 7,
					want: 2,
				},
			},
		},
		{
			grouping: "no underground",
			tests: []oneTC{
				{
					x: 92, y: 94, floor: 8,
					want: 0,
				},
				{
					x: 100, y: 100, floor: 8,
					want: 0,
				},
			},
		},
		{
			grouping: "top left floor up ok",
			tests: []oneTC{
				{
					x: 92, y: 94, floor: 6,
					want: 0,
				},
				{
					x: 93, y: 94, floor: 6,
					want: 0,
				},
				{
					x: 92, y: 95, floor: 6,
					want: 0,
				},
				{
					x: 93, y: 95, floor: 6,
					want: 2,
				},
			},
		},
	}

	for _, tcg := range tcs {
		t.Run(tcg.grouping, func(t *testing.T) {
			for _, tc := range tcg.tests {
				t.Run(fmt.Sprintf("%d.%d.%d", tc.x, tc.y, tc.floor), func(t *testing.T) {
					tile, err := ds.GetMapTile(tc.x, tc.y, tc.floor)
					if err != nil {
						t.Fatalf("error getting tile %d %d %d", tc.x, tc.y, tc.floor)
					}
					if got := tile.(*accessCountingTile).accessCount; got != tc.want {
						t.Errorf("@ %d,%d,%d access count want %d, got %d", tc.x, tc.y, tc.floor, tc.want, got)
					}
				})
			}
		})
	}
}

func TestMapDescriptionRangeMoveNorth(t *testing.T) {
	playerID := CreatureID(123)
	gws := &GameworldServer{
		things: LoadThingsForTest(t),
	}
	gwConn := &GameworldConnection{}
	gwConn.clientVersion = 854
	gwConn.server = gws
	//gwConn.conn = conn
	//gwConn.key = key
	gwConn.id = GameworldConnectionID(playerID)

	ds := &accessCountingMapDataSource{
		mapDataSource: *(NewMapDataSource().(*mapDataSource)),
	}
	ds.mapTileGenerator = func(x, y uint16, z uint8) (MapTile, error) {
		mt, err := generateAccessCountingMapTile(x, y, z)
		if err != nil {
			return nil, err
		}
		mt.(*accessCountingTile).tst = t
		return mt, err
	}
	gws.SetMapDataSource(ds)

	ds.AddCreature(&creature{
		id:   123,
		pos:  tnet.Position{X: 100, Y: 91, Floor: 7},
		dir:  things.CreatureDirectionSouth,
		look: 128,
		col:  [4]things.OutfitColor{0x0a, 0x0a, 0x0a, 0x0a},
	})

	msg := tnet.NewMessage()
	err := gwConn.playerMoveNorthImpl(msg)
	if err != nil {
		t.Errorf("failed to send map: %v", err)
	}
	type oneTC struct {
		x, y  uint16
		floor uint8
		want  int
	}
	tcs := []struct {
		grouping string
		tests    []oneTC
	}{
		{
			grouping: "left edge",
			tests: []oneTC{
				{
					x: 91, y: 84, floor: 7,
					want: 0,
				},
				{
					x: 92, y: 83, floor: 7,
					want: 0,
				},
				{
					x: 92, y: 84, floor: 7,
					want: 2,
				},
				{
					x: 92, y: 85, floor: 7,
					want: 0,
				},
			},
		},
		{
			grouping: "right edge",
			tests: []oneTC{
				{
					x: 110, y: 84, floor: 7,
					want: 0,
				},
				{
					x: 109, y: 83, floor: 7,
					want: 0,
				},
				{
					x: 109, y: 84, floor: 7,
					want: 2,
				},
				{
					x: 109, y: 85, floor: 7,
					want: 0,
				},
			},
		},
		{
			grouping: "no underground",
			tests: []oneTC{
				{
					x: 91, y: 84, floor: 8,
					want: 0,
				},
			},
		},
		{
			grouping: "left floor up ok",
			tests: []oneTC{
				{
					x: 92, y: 84, floor: 6,
					want: 0,
				},
				{
					x: 93, y: 84, floor: 6,
					want: 0,
				},
				{
					x: 92, y: 85, floor: 6,
					want: 0,
				},
				{
					x: 93, y: 85, floor: 6,
					want: 1,
				},
			},
		},
	}

	for _, tcg := range tcs {
		t.Run(tcg.grouping, func(t *testing.T) {
			for _, tc := range tcg.tests {
				t.Run(fmt.Sprintf("%d.%d.%d", tc.x, tc.y, tc.floor), func(t *testing.T) {
					tile, err := ds.GetMapTile(tc.x, tc.y, tc.floor)
					if err != nil {
						t.Fatalf("error getting tile %d %d %d", tc.x, tc.y, tc.floor)
					}
					if got := tile.(*accessCountingTile).accessCount; got != tc.want {
						t.Errorf("@ %d,%d,%d access count want %d, got %d", tc.x, tc.y, tc.floor, tc.want, got)
					}
				})
			}
		})
	}
}
