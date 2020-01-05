package gameworld

import (
	"fmt"
	"testing"

	tnet "badc0de.net/pkg/go-tibia/net"
)

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
	if t.tst != nil { t.tst.Logf("@ %d,%d,%d accessed idx %d", t.x, t.y, t.z, idx) } 
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
		mt, err := generateAccessCountingMapTile(x,y,z)
		if err != nil {return nil, err}
		mt.(*accessCountingTile).tst = t
		return mt, err
	}
	gws.SetMapDataSource(ds)

	ds.AddCreature(&creature{
		id: 123,
		x:  100,
		y:  100,
		z:  7,
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

