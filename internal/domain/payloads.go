package domain

// CredentialPayload stores login/password data.
type CredentialPayload struct {
	Login    string
	Password string
}

// RecordType returns the payload type.
func (CredentialPayload) RecordType() RecordType { return RecordTypeCredential }

// TextPayload stores arbitrary text data.
type TextPayload struct {
	Text string
}

// RecordType returns the payload type.
func (TextPayload) RecordType() RecordType { return RecordTypeText }

// BinaryPayload stores arbitrary binary data.
type BinaryPayload struct {
	Data []byte
}

// RecordType returns the payload type.
func (BinaryPayload) RecordType() RecordType { return RecordTypeBinary }

// BankCardPayload stores bank card details.
type BankCardPayload struct {
	Cardholder string
	Number     string
	ExpiresAt  string
	CVV        string
}

// RecordType returns the payload type.
func (BankCardPayload) RecordType() RecordType { return RecordTypeBankCard }
