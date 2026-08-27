package domain

type AuthorizePaymentRequest struct {
	AccountID string `json:"account_id" binding:"required,uuid"`
	Amount    int64  `json:"amount" binding:"required,gt=0"`
	Currency  string `json:"currency" binding:"required,len=3,alpha"`
}
