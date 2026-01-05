package domain

type Money struct {
	Currency string
	Amount   int64
}

type QuoteLine struct {
	ProductID string
	Name      string
	Quantity  int64
	UnitPrice Money
	LineTotal Money
}

type Quote struct {
	Lines []QuoteLine
	Total Money
}
