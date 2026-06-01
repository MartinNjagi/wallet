package data

var RedisKeyUserPermissions = "user:permissions:%d" // user_id

// Allowed reference prefixes to namespace operations strictly
const (
	RefPrefixMpesa = "MPESA_"
	RefPrefixSMS   = "SMS_"
	RefPrefixAdj   = "ADJ_"
	RefPrefixRev   = "REV_"
)

type WalletAction string

const (
	WalletActionCredit WalletAction = "credit"
	WalletActionDebit  WalletAction = "debit"
)

const DefaultCurrency = "KES"
