package oauth

// Scope is an Etsy OAuth 2.0 permission scope.
type Scope string

const (
	ScopeAddressRead       Scope = "address_r"
	ScopeAddressWrite      Scope = "address_w"
	ScopeEmailRead         Scope = "email_r"
	ScopeListingsDelete    Scope = "listings_d"
	ScopeListingsRead      Scope = "listings_r"
	ScopeListingsWrite     Scope = "listings_w"
	ScopeProfileRead       Scope = "profile_r"
	ScopeProfileWrite      Scope = "profile_w"
	ScopeShopsRead         Scope = "shops_r"
	ScopeShopsWrite        Scope = "shops_w"
	ScopeTransactionsRead  Scope = "transactions_r"
	ScopeTransactionsWrite Scope = "transactions_w"
)
