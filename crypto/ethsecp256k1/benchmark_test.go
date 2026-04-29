package ethsecp256k1

import (
	"fmt"
	"testing"
)

func BenchmarkGenerateKey(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GenerateKey()
	}
}

func BenchmarkPubKey_VerifySignature(b *testing.B) {
	privKey := GenerateKey()
	pubKey := privKey.PubKey()

	// Precompute messages and signatures
	const numSamples = 100
	msgs := make([][]byte, numSamples)
	sigs := make([][]byte, numSamples)
	for i := 0; i < numSamples; i++ {
		msgs[i] = fmt.Appendf(nil, "%10d", i)
		sig, err := privKey.Sign(msgs[i])
		if err != nil {
			b.Fatal(err)
		}
		sigs[i] = sig
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		pubKey.VerifySignature(msgs[i%numSamples], sigs[i%numSamples])
	}
}
