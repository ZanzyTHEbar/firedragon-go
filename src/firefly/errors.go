package firefly

import (
	"github.com/ZanzyTHEbar/errbuilder-go"
)

// Error codes specific to Firefly operations
const (
	ErrInvalidTransaction = "invalid_transaction"
	ErrInvalidAccount     = "invalid_account"
	ErrInvalidCategory    = "invalid_category"
	ErrInvalidBudget      = "invalid_budget"
	ErrInvalidAttachment  = "invalid_attachment"
	ErrAPIFailure         = "api_failure"
	ErrNotFound           = "not_found"
	ErrDuplicateEntry     = "duplicate_entry"
	ErrRateLimit          = "rate_limit"
)

// TransactionValidationErr returns a validation error for transactions
func TransactionValidationErr(errors errbuilder.ErrorMap) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeInvalidArgument).
		WithMsg("Invalid Transaction Data").
		WithDetails(errbuilder.NewErrDetails(errors))
}

// AccountValidationErr returns a validation error for accounts
func AccountValidationErr(errors errbuilder.ErrorMap) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeInvalidArgument).
		WithMsg("Invalid Account Data").
		WithDetails(errbuilder.NewErrDetails(errors))
}

// CategoryValidationErr returns a validation error for categories
func CategoryValidationErr(errors errbuilder.ErrorMap) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeInvalidArgument).
		WithMsg("Invalid Category Data").
		WithDetails(errbuilder.NewErrDetails(errors))
}

// BudgetValidationErr returns a validation error for budgets
func BudgetValidationErr(errors errbuilder.ErrorMap) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeInvalidArgument).
		WithMsg("Invalid Budget Data").
		WithDetails(errbuilder.NewErrDetails(errors))
}

// AttachmentValidationErr returns a validation error for attachments
func AttachmentValidationErr(errors errbuilder.ErrorMap) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeInvalidArgument).
		WithMsg("Invalid Attachment Data").
		WithDetails(errbuilder.NewErrDetails(errors))
}

// APIErr returns an error for API failures
func APIErr(msg string, err error) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeInternal).
		WithMsg(msg).
		WithCause(err)
}

// NotFoundErr returns a not found error
func NotFoundErr(resourceType string, err error) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeNotFound).
		WithMsg(resourceType + " Not Found").
		WithCause(err)
}

// DuplicateErr returns a duplicate entry error
func DuplicateErr(resourceType string, err error) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeAlreadyExists).
		WithMsg("Duplicate " + resourceType).
		WithCause(err)
}

// RateLimitErr returns a rate limit error
func RateLimitErr(err error) error {
	return errbuilder.NewErrBuilder().
		WithCode(errbuilder.CodeResourceExhausted).
		WithMsg("Rate Limit Exceeded").
		WithCause(err)
}

// validateTransaction validates a transaction and returns an error map
func validateTransaction(tx TransactionModel) errbuilder.ErrorMap {
	var errs errbuilder.ErrorMap

	if tx.Amount <= 0 {
		errs.Set("amount", "Amount must be greater than 0")
	}
	if tx.Currency == "" {
		errs.Set("currency", "Currency is required")
	}
	if tx.Description == "" {
		errs.Set("description", "Description is required")
	}
	if tx.TransType == "" {
		errs.Set("type", "Transaction type is required")
	}
	if tx.Date.IsZero() {
		errs.Set("date", "Date is required")
	}

	return errs
}

// validateAccount validates an account and returns an error map
func validateAccount(account AccountModel) errbuilder.ErrorMap {
	var errs errbuilder.ErrorMap

	if account.Name == "" {
		errs.Set("name", "Name is required")
	}
	if account.Type == "" {
		errs.Set("type", "Account type is required")
	}
	if account.Currency == "" {
		errs.Set("currency", "Currency is required")
	}

	return errs
}

// validateCategory validates a category and returns an error map
func validateCategory(category CategoryModel) errbuilder.ErrorMap {
	var errs errbuilder.ErrorMap

	if category.Name == "" {
		errs.Set("name", "Name is required")
	}

	return errs
}

// validateBudget validates a budget and returns an error map
func validateBudget(budget BudgetModel) errbuilder.ErrorMap {
	var errs errbuilder.ErrorMap

	if budget.Name == "" {
		errs.Set("name", "Name is required")
	}
	if budget.AutoBudgetAmount != nil && budget.AutoBudgetPeriod == nil {
		errs.Set("auto_budget_period", "Auto budget period is required when amount is set")
	}
	if budget.AutoBudgetPeriod != nil && budget.AutoBudgetAmount == nil {
		errs.Set("auto_budget_amount", "Auto budget amount is required when period is set")
	}

	return errs
}

// validateBudgetLimit validates a budget limit and returns an error map
func validateBudgetLimit(limit BudgetLimitModel) errbuilder.ErrorMap {
	var errs errbuilder.ErrorMap

	if limit.Amount == "" {
		errs.Set("amount", "Amount is required")
	}
	if limit.Period == "" {
		errs.Set("period", "Period is required")
	}
	if limit.Start.IsZero() {
		errs.Set("start", "Start date is required")
	}
	if limit.End.IsZero() {
		errs.Set("end", "End date is required")
	}
	if limit.End.Before(limit.Start) {
		errs.Set("end", "End date must be after start date")
	}

	return errs
}

// validateAttachment validates an attachment and returns an error map
func validateAttachment(filename string, file []byte, title string) errbuilder.ErrorMap {
	var errs errbuilder.ErrorMap

	if filename == "" {
		errs.Set("filename", "Filename is required")
	}
	if len(file) == 0 {
		errs.Set("file", "File content is required")
	}
	if title == "" {
		errs.Set("title", "Title is required")
	}

	return errs
}
