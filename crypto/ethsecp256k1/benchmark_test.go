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

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; b.Loop(); i++ {
		b.StopTimer()
		msg := fmt.Appendf(nil, "%10d", i)
		sig, err := privKey.Sign(msg)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		pubKey.VerifySignature(msg, sig)
	}
}
