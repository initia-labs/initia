package types

const (
	TxSignatureContextKey = "tx_signature"
)

type AbstractionData struct {
	FunctionInfo FunctionInfo        `json:"function_info"`
	AuthData     AbstractionAuthData `json:"auth_data"`
}

type FunctionInfo struct {
	ModuleAddress string `json:"module_address"`
	ModuleName    string `json:"module_name"`
	FunctionName  string `json:"function_name"`
}

type AbstractionAuthData struct {
	V1          *V1AuthData          `json:"V1,omitempty"`
	DerivableV1 *DerivableV1AuthData `json:"DerivableV1,omitempty"`
}

type V1AuthData struct {
	SigningMessageDigest []uint8 `json:"signing_message_digest"`
	Authenticator        []uint8 `json:"authenticator"`
}

type DerivableV1AuthData struct {
	SigningMessageDigest []uint8 `json:"signing_message_digest"`
	AbstractSignature    []uint8 `json:"abstract_signature"`
	AbstractPublicKey    []uint8 `json:"abstract_public_key"`
}
