package decimalx

import "github.com/shopspring/decimal"

var Zero = decimal.Zero

func NewFromFloat(value float64) decimal.Decimal {
	return decimal.NewFromFloat(value)
}

func NewFromString(value string) (decimal.Decimal, error) {
	return decimal.NewFromString(value)
}

func Money(value string) decimal.Decimal {
	d, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero
	}
	return d
}
