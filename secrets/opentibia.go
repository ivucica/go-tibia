package secrets

import (
	"crypto/rsa"
	"fmt"
	"github.com/golang/glog"
	"math/big"
)

var (
	bigOne = big.NewInt(1)
	// RSA private key used by OpenTibia server to decrypt content encrypted with the corresponding public key.
	OpenTibiaPrivateKey rsa.PrivateKey
)

func init() {
	initOpenTibiaPK()
}

func initOpenTibiaPK() error {
	p := "14299623962416399520070177382898895550795403345466153217470516082934737582776038882967213386204600674145392845853859217990626450972452084065728686565928113"
	q := "7630979195970404721891201847792002125535401292779123937207447574596692788513647179235335529307251350570728407373705564708871762033017096809910315212884101"
	pB, ok := new(big.Int).SetString(p, 10)
	if !ok {
		glog.Errorln("initOpenTibiaPK(): invalid p")
		return fmt.Errorf("login: invalid p")
	}
	qB, ok := new(big.Int).SetString(q, 10)
	if !ok {
		glog.Errorln("initOpenTibiaPK(): invalid q")
		return fmt.Errorf("login: invalid q")
	}

	p1 := new(big.Int).Sub(pB, bigOne)
	q1 := new(big.Int).Sub(qB, bigOne)

	p1q1 := new(big.Int).Mul(p1, q1)
	pubK := rsa.PublicKey{
		E: 65537,
		N: new(big.Int).Mul(pB, qB),
	}
	pk := rsa.PrivateKey{
		Primes:    []*big.Int{pB, qB},
		PublicKey: pubK,
		D:         new(big.Int).ModInverse(big.NewInt(int64(pubK.E)), p1q1),
	}

	pk.Precompute()
	OpenTibiaPrivateKey = pk

	return nil
}
