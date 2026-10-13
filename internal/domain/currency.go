package domain

// ISO 4217 minor-unit exponents (0 = JPY, 3 = KWD ...)
var currencyExponent = map[string]int{
	"USD": 2, "EUR": 2, "GBP": 2, "NGN": 2,
	"JPY": 0, "KRW": 0,
	"KWD": 3, "BHD": 3, "OMR": 3,
}

// currencies the upstream bank actually handles (it ignores the field and always uses USD)
var bankSupported = map[string]bool{"USD": true}

func CurrencyExponent(code string) (int, bool) { e, ok := currencyExponent[code]; return e, ok }
func IsKnownCurrency(code string) bool         { _, ok := currencyExponent[code]; return ok }
func IsSupportedCurrency(code string) bool     { return bankSupported[code] }
