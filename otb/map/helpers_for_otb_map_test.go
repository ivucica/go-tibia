package otbm

import (
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/gameworld/gwmap"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/things"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
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

// This test code is testing gameworld, but to load the map, we would need to
// import the OTBM loader.
//
// But OTBM loader used to import gameworld, due to some map types being here.
//
// So the actual test is invoked from OTBM loader until this is all
// disentangled.
func mapDescriptionEncoding_Test(t *testing.T, ds1, ds2 gwmap.MapDataSource) {
	t.Run("mapDescriptionEncodingInitialSpawn", func(t *testing.T) {
		mapDescriptionEncodingInitialSpawn(t, ds1)
	})

	t.Run("mapDescriptionEncodingInitialSpawnMoveNorth", func(t *testing.T) {
		mapDescriptionEncodingMoveNorth(t, ds2)
	})

}

func mapDescriptionEncodingInitialSpawn(t *testing.T, ds gwmap.MapDataSource) {
	playerID := gwmap.CreatureID(0x100003e9) // e9030010
	gws := &gameworld.GameworldServer{}
	gws.SetThings(LoadThingsForTest(t))
	gwConn := &gameworld.GameworldConnection{}
	gwConn.TestOnly_Setter(854, gws, gameworld.GameworldConnectionID(playerID))
	//gwConn.conn = conn
	//gwConn.key = key

	gws.SetMapDataSource(ds)

	ds.AddCreature(gameworld.BakeTestOnlyCreature(
		/*id: */ playerID,
		/*pos: */ tnet.Position{
			X:     100,
			Y:     100,
			Floor: 7,
		},
		/*dir: */ 0,
		/*look: */ 0x88,
		/*col: */ [4]things.OutfitColor{0x0a, 0x0a, 0x0a, 0x0a},
	))

	msg := tnet.NewMessage()
	err := gwConn.TestOnly_InitialAppearMap(msg)
	if err != nil {
		t.Errorf("failed to send map: %v", err)
	}

	blobs := []struct {
		name   string
		hexa   string
		hexa14 [14]string
		blob   []byte
	}{

		/*
			{
				name: "player id introduction"
				hexa: `0a`,
			},
			{
				name: "player id",
				hexa: `e9030010`,
			},
			{
				name: "draw speed and can send bug reports",
				hexa: `320000`,
			},
		*/
		{
			name: "teleport",
			hexa: `64`,
		},
		{
			name: "player location",
			hexa: `6400640007`,
		},
		{
			name:   "leftmost (92:94:7 - 92:107:7)",
			hexa14: [14]string{`a3111d1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `a3111c1200ff`, `af111c1200ff`, `a3111f1200ff`},
		},
		{
			name:   "second (93:94:7 - 93:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `a20100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a71100ff`, `a20100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "third (94:94:7 - 94:107:7)",
			hexa14: [14]string{`a311191200ff`, `a20100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a61100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a20100ff`, `a3111b1200ff`},
		},
		{
			name:   "fourth (95:94:7 - 95:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `a81100ff`, `a31100ff`, `a31100ff`, `b21100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "fifth: (96:94:7 - 96:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a41100ff`, `a31100ff`, `a31100ff`, `ae1100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a61100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "sixth (97:94:7 - 97:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `a31100ff`, `a3112e1a00ff`, `a311281a00ff`, `a3112c1a00ff`, `a31100ff`, `ab112e1a00ff`, `a711281a00ff`, `a3112c1a00ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "seventh (98:94:7 - 98:107:7)",
			hexa14: [14]string{`a311191200ff`, `ad0100ff`, `a31100ff`, `a311291a00ff`, `1c1a00ff`, `a311271a00ff`, `a31100ff`, `a311291a00ff`, `200300ff`, `a311271a00ff`, `a31100ff`, `a31100ff`, `ad0100ff`, `a3111b1200ff`},
		},
		{
			name:   "eighth (99:94:7 - 99:107:7)",
			hexa14: [14]string{`a311191200ff`, `ad0100ff`, `a61100ff`, `a3112d1a00ff`, `a3112a1a00ff`, `a311db032b1a00ff`, `a311d70300ff`, `a311d9032d1a00ff`, `a3112a1a00ff`, `ad112b1a00ff`, `ae1100ff`, `a31100ff`, `ad0100ff`, `a3111b1200ff`},
		},
		{
			name: "CENTER TOP (100:94:7 - 100:99:7)",
			hexa: `a311191200ff` + `ad0100ff` + `a31100ff` + `a31100ff` + `a31100ff` + `a311d40300ff`,
		},
		{
			name: "PLAYER TILE (100:100:7)",
			hexa: (`e700` + // [0 + 2] ground item (sand)
				`6100` + // [2 + 2] new creature
				`00000000` + // [4 + 4] remove nobody (id 00 00 00 00)
				`e9030010` + // [8 + 4] creature id (player)
				`0e0044656d6f20436861726163746572` + // [12 + 16] "Demo Character"
				`64` + // [28 + 1] health
				`00` + // [29 + 1] direction
				`88000a0a0a` + // [30 + 5] outfit type + colors
				`0a00` + // [35 + 2] looktype extended
				`0000` + // [37 + 2] lightlevel + color
				`8403` + // [39 + 2] step speed
				`0000` + // [41 + 2] skull, party shield
				`00` + // [43 + 1] can send war emblem
				`01` + // [44 + 1] player can walk through
				`00ff`), // [45 + 2] end tile

		},
		{
			name: "CENTER BOTTOM (100:101:7 - 100:107:7)",
			hexa: `a311d30300ff` + `a31100ff` + `a31100ff` + `a31100ff` + `a31100ff` + `ad0100ff` + `a3111b1200ff`,
		},
		{
			name:   "tenth (101:94:7 - 101:107:7)",
			hexa14: [14]string{`a311191200ff`, `ad0100ff`, `a31100ff`, `a3112e1a00ff`, `a311281a00ff`, `af11da032c1a00ff`, `a311d50300ff`, `a311d8032e1a00ff`, `a311281a00ff`, `a3112c1a00ff`, `af1100ff`, `a31100ff`, `ad0100ff`, `a3111b1200ff`},
		},
		{
			name:   "eleventh (102:94:7 - 102:107:7)",
			hexa14: [14]string{`a311191200ff`, `ad0100ff`, `a31100ff`, `a511291a00ff`, `200300ff`, `a311271a00ff`, `a31100ff`, `a311291a00ff`, `1b1a00ff`, `a311271a00ff`, `a31100ff`, `a31100ff`, `ad0100ff`, `a3111b1200ff`},
		},
		{
			name:   "twelfth (103:94:7 - 103:107:7)",
			hexa14: [14]string{`ac11191200ff`, `a31100ff`, `a31100ff`, `a3112d1a00ff`, `a3112a1a00ff`, `a3112b1a00ff`, `a31100ff`, `a3112d1a00ff`, `a3112a1a00ff`, `a3112b1a00ff`, `a31100ff`, `a31100ff`, `ad1100ff`, `aa111b1200ff`},
		},
		{
			name:   "thirteenth (104:94:7 - 104:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `a31100ff`, `a31100ff`, `ad1100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "fifteenth (105:94:7 - 105:107:7)",
			hexa14: [14]string{`a311191200ff`, `a61100ff`, `b21100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `ac1100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "sixteenth (106:94:7 - 106:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `aa1100ff`, `a31100ff`, `a31100ff`, `ac1100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "seventeenth (107:94:7 - 107:107:7)",
			hexa14: [14]string{`a311191200ff`, `a20100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a61100ff`, `a31100ff`, `a31100ff`, `ab1100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a20100ff`, `a6111b1200ff`},
		},
		{
			name:   "eighteenth (108:94:7 - 108:107:7)",
			hexa14: [14]string{`a311191200ff`, `a31100ff`, `a20100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `b21100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a31100ff`, `a20100ff`, `a31100ff`, `a3111b1200ff`},
		},
		{
			name:   "nineteenth (109:94:7 - 109:107:7)",
			hexa14: [14]string{`a3111e1200ff`, `a3111a1200ff`, `ac111a1200ff`, `a3111a1200ff`, `af111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a3111a1200ff`, `a311201200ff`},
		},
		{
			name: "floor 6 leftmost (93:95:6 - 93:108:6)",
			hexa: `1f030cff` + `1f0300ff`, // 031f = 670 = snow, skip 12 tiles, 031f = 670 = snow
		},
		{
			name: "floor 6 second (94:95:6 + 2 tiles -> 94:97:6 + 1+10 tiles -> 94:107:6 + 1+57(4*14) tiles -> 99:95:6)",
			hexa: `a70600ff` + `66030aff` + `660339ff`, // 06a7 = 1703 = sand 94:95:6, 0366 = 870 = cobbled pavement 94:96:6 + skip 10 tiles, 0366 = 870 = cobbled pavement 94:107:6
		},
		{
			name: "floor 6 third (99:95:6 -> 99:99:6 -> 99:103:6 -> 99:108:6)",
			hexa: (`980103ff` + // 0198 = 408 = wooden floor, then skip 1+3 tiles
				`9f0c03ff` + // 0c9f = 3231 = gemmed lamp, then skip 1+3 tiles
				`9f0c04ff` + // 0c9f = 3231 = gemmed lamp, then skip 1+4 tiles
				`980100ff`), // 0198 = 408 = wooden floor, then skip 1 tile
		},
		{
			name: "floor 6 fourth (100:95:6 -> 100:108:6 -> 101:95:6 -> 101:108:6 -> 102:95:6 -> 102:108:6)",
			hexa: (`98010cff` + //  100:95:6  0198 = 408 = wooden floor, then skip 1+12 tiles
				`980100ff` + // 100:108:6 0198 = 408 = wooden floor, then skip 1 tiles
				`98010cff` + // 101:95:6  0198 = 408 = wooden floor, then skip 1+12 tiles
				`980100ff` + // 101:108:6 0198 = 408 = wooden floor, then skip 1 tiles
				`98010cff` + // 102:95:6  0198 = 408 = wooden floor, then skip 1+12 tiles
				`980100ff`), // 102:108:6 0198 = 408 = wooden floor, then skip 1 tiles
		},
		{
			name: "floor 6 fifth (103:95:6 -> 103:108:6)",
			hexa: (`980103ff` + //  103:95:6  0198 = 408 = wooden floor, then skip 1+3 tiles
				`9f0c03ff` + // 103:99:6  0c9f = 3231 = gemmed lamp, then skip 1+3 tiles
				`9f0c04ff` + // 103:103:6 0c9f = 3231 = gemmed lamp, then skip 1+4 tiles
				`980147ff`), // 103:108:6 0198 = 408 = wooden floor, then skip 1+71 tiles (5 columns/X + 1 row/Y + 1 row/Y => 108:110[!]:6 => 109:96:6)
		},
		{
			name: "floor 6 seventh (109:95:6 -> 110:108:6)",
			hexa: (`66030aff` + // 109:96:6  0366 = 870 = cobbled pavement, then skip 1+10 tiles
				`660301ff` + // 109:107:6 0366 = 870 = cobbled pavement, then skip 1+1 tiles
				`bd190cff` + // 110:95:6 19bd = 6589 = snow, then skip 1+12 tiles
				`bb19ffff`), // 110:108:6 19bd = 6589 = snow, then skip 1+255 tiles
		},
		{
			name: "rest",
			hexa: `ffffffffffffffffe8ff`,
		},
	}
	wantLen := 0
	for idx := range blobs {
		if blobs[idx].hexa == "" {
			blobs[idx].hexa = strings.Join(blobs[idx].hexa14[:], "")
		}
		blob, err := hex.DecodeString(blobs[idx].hexa)
		if err != nil {
			t.Fatalf("cannot decode golden string for %v: %v", blobs[idx].name, err)
		}
		blobs[idx].blob = blob
		wantLen += len(blob)
	}

	if wantLen != msg.Len() {
		t.Errorf("wrong message size: got %d, want %d", msg.Len(), wantLen)
	}

	var totalIdx int
	for blobIdx, blob := range blobs {
		t.Run(fmt.Sprintf("%d/%s", blobIdx, blob.name), func(t *testing.T) {
			for idx, c := range blob.blob {
				b, err := msg.ReadByte()
				totalIdx++
				if err != nil {
					t.Errorf("wanted to read byte %02x at idx %d (blob %d/%s: byte %d), got %v", b, totalIdx, blobIdx, blob.name, idx, err)
					break
				}
				if b != c {
					t.Errorf("diff at byte %d (blob %d/%s: byte %d): got %02x, want %02x", totalIdx, blobIdx, blob.name, idx, b, c)
					break
				}
			}
		})
	}
}

func mapDescriptionEncodingMoveNorth(t *testing.T, ds gwmap.MapDataSource) {
	playerID := gwmap.CreatureID(0x100003e9) // e9030010
	gws := &gameworld.GameworldServer{}
	gws.SetThings(LoadThingsForTest(t))
	gwConn := &gameworld.GameworldConnection{}
	gwConn.TestOnly_Setter(854, gws, gameworld.GameworldConnectionID(playerID))
	//gwConn.conn = conn
	//gwConn.key = key

	gws.SetMapDataSource(ds)

	ds.AddCreature(gameworld.BakeTestOnlyCreature(
		/*id: */ playerID,
		/*pos: */ tnet.Position{
			X:     100,
			Y:     91,
			Floor: 7,
		},
		/*dir: */ 0,
		/*look: */ 0x88,
		/*col: */ [4]things.OutfitColor{0x0a, 0x0a, 0x0a, 0x0a},
	))

	// First, sanity checking tile on top left: 92,84,7 (which should be 980100ff -- just item 405):
	if topLeftTile, err := ds.GetMapTile(92, 84, 7); err != nil {
		t.Errorf("error fetching topleft tile: %v", err)
	} else {
		if item, err := topLeftTile.GetItem(0); err != nil {
			t.Errorf("error fetching bottom item at topleft tile of %v: %v", ds, err)
		} else {
			if item.GetServerType() != 405 {
				t.Errorf("wrong item at topleft tile of %v; got %d, want 405", ds, item.GetServerType())
			}
		}
	}

	msg := tnet.NewMessage()
	if err := gwConn.TestOnly_PlayerMoveNorthImpl(msg); err != nil {
		t.Errorf("%v", err)
	}

	t.Logf("%s", hex.EncodeToString(msg.Bytes()))

	// 6d
	//   6400 5b00 07 01
	//   6400 5a00 07
	// 65
	// got:
	//   02ff a311010508ff a311010508ff [-3]
	//[+3] 5c0b010508ff 5c0b010506ff [-1]
	//[+1] 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 91045aff
	// want:
	//   980100ff 980100ff a311010500ff a311010500ff 980100ff 980100ff 980100ff 980100ff 980100ff 980100ff 980100ff a311010500ff a311010500ff 980100ff 980100ff 980100ff 980100ff 980100ff
	//   980100ff 9801010500ff 01055c0b00ff 980100ff 980100ff 980100ff 980100ff 980100ff 980100ff 980100ff 9801010500ff 01055c0b00ff 980100ff 980100ff 980100ff 980100ff 980100ff 980100ff
	//   910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 910400ff 91045aff
	//t.Errorf("fake fail")
}
