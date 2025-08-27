package keys

type PubKey interface {
	Verify(message []byte, signature []byte) error
}
