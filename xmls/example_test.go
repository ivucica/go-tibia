package xmls

import (
	"bytes"
	"fmt"
)

func Example() {
	o := bytes.NewReader([]byte(`<?xml version="1.0"?>
<outfits>
	<outfit id="1" premium="0">
		<list type="female" looktype="136" name="Citizen"/>
		<list type="male" looktype="128" name="Citizen"/>
	</outfit>
</outfits>`))
	outfits, err := ReadOutfits(o)
	if err != nil {
		panic(err)
	}

	fmt.Println(outfits.Outfit[0].List[0].Name)
	// Output:
	// Citizen
}
