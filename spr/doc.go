// Package spr implements a reader for individual sprites in a Tibia.spr files.
//
// A higher level implementation needs to be used together with the dataset
// information on a thing's graphics layout and sprites in order to actually
// construct a full recognizable image.
//
// Since .pic format also uses the same encoder, it can be used as a basis for
// a .pic decoder.
package doc
