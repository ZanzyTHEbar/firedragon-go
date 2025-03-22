package firefly

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/oapi-codegen/runtime/types"
)

// PiggyBankModel represents a piggy bank in our domain model
type PiggyBankModel struct {
	ID                    string
	Name                  string
	AccountID            string
	TargetAmount         string
	CurrentAmount        string
	StartDate           *time.Time
	TargetDate          *time.Time
	Notes               *string
	Order               *int32
	Active              bool
	Percentage          float32
	CurrencyCode        string
	CurrencySymbol      string
	LeftToSave          string
	SavePerMonth        string
	ObjectGroupID       *string
	ObjectGroupTitle    *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// PiggyBankEventModel represents an event in a piggy bank's history
type PiggyBankEventModel struct {
	ID                    string
	PiggyBankID          string
	TransactionJournalID string
	Amount               string
	CurrencyCode         string
	CurrencySymbol       string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

// DataType represents the type of data to export or destroy
type DataType string

const (
	DataTypeAccounts     DataType = "accounts"
	DataTypeBudgets      DataType = "budgets"
	DataTypeCategories   DataType = "categories"
	DataTypeTransactions DataType = "transactions"
)

// ExportFormat represents the format for data export
type ExportFormat string

const (
	ExportFormatCSV ExportFormat = "csv"
)

// BillModel represents a bill in our domain model
type BillModel struct {
	ID                    string
	Name                  string
	AmountMin            string
	AmountMax            string
	Date                 time.Time
	EndDate              *time.Time
	ExtensionDate        *time.Time
	CurrencyCode         *string
	CurrencyID           *string
	CurrencySymbol       *string
	CurrencyDecimalPlaces *int32
	NativeAmountMax      *string
	Active               *bool
	Notes                *string
	ObjectGroupID        *string
	ObjectGroupTitle     *string
	CreatedAt            *time.Time
	UpdatedAt            *time.Time
}

// CreatePiggyBank creates a new piggy bank
func (c *FireflyClient) CreatePiggyBank(piggyBank PiggyBankModel) error {
	// Validate piggy bank
	if errs := validatePiggyBank(piggyBank); errs != nil {
		return ValidationErr("PiggyBank", errs)
	}

	ctx := context.Background()

	// Create piggy bank request
	request := PiggyBankStore{
		Name:         piggyBank.Name,
		TargetAmount: &piggyBank.TargetAmount,
		StartDate:    dateToAPIDate(piggyBank.StartDate),
		TargetDate:   dateToAPIDate(piggyBank.TargetDate),
		Notes:        piggyBank.Notes,
		Order:        piggyBank.Order,
		Active:       &piggyBank.Active,
	}

	// Call the API
	resp, err := c.clientAPI.StorePiggyBankWithResponse(ctx, &StorePiggyBankParams{}, request)
	if err != nil {
		return APIErr("Failed to create piggy bank", err)
	}

	// Check response
	if resp.StatusCode() == http.StatusConflict {
		return DuplicateErr("PiggyBank", fmt.Errorf("piggy bank already exists"))
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return RateLimitErr(fmt.Errorf("rate limit exceeded"))
	}
	if resp.StatusCode() != http.StatusOK && resp.StatusCode() != http.StatusCreated {
		return APIErr("Failed to create piggy bank", fmt.Errorf("unexpected status: %s", resp.Status()))
	}

	return nil
}

// GetPiggyBank retrieves a single piggy bank by ID
func (c *FireflyClient) GetPiggyBank(id string) (*PiggyBankModel, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.GetPiggyBankWithResponse(ctx, id, &GetPiggyBankParams{})
	if err != nil {
		return nil, APIErr("Failed to get piggy bank", err)
	}

	// Check response
	if resp.StatusCode() == http.StatusNotFound {
		return nil, NotFoundErr("PiggyBank", fmt.Errorf("piggy bank not found: %s", id))
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return nil, RateLimitErr(fmt.Errorf("rate limit exceeded"))
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, APIErr("Failed to get piggy bank", fmt.Errorf("unexpected status: %s", resp.Status()))
	}

	// Convert API response to PiggyBankModel
	if resp.HTTPResponse == nil || len(resp.Body) == 0 {
		return nil, APIErr("No piggy bank data found", fmt.Errorf("empty response"))
	}

	var apiResp PiggyBankSingle
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return nil, APIErr("Failed to parse piggy bank response", err)
	}

	piggyBank := &PiggyBankModel{
		ID:               apiResp.Data.Id,
		Name:             apiResp.Data.Attributes.Name,
		TargetAmount:     stringValue(apiResp.Data.Attributes.TargetAmount),
		CurrentAmount:    stringValue(apiResp.Data.Attributes.CurrentAmount),
		StartDate:        apiDateToTime(apiResp.Data.Attributes.StartDate),
		TargetDate:       apiDateToTime(apiResp.Data.Attributes.TargetDate),
		Notes:            apiResp.Data.Attributes.Notes,
		Order:            apiResp.Data.Attributes.Order,
		Active:           boolValue(apiResp.Data.Attributes.Active),
		Percentage:       float32Value(apiResp.Data.Attributes.Percentage),
		CurrencyCode:     stringValue(apiResp.Data.Attributes.CurrencyCode),
		CurrencySymbol:   stringValue(apiResp.Data.Attributes.CurrencySymbol),
		LeftToSave:       stringValue(apiResp.Data.Attributes.LeftToSave),
		SavePerMonth:     stringValue(apiResp.Data.Attributes.SavePerMonth),
		ObjectGroupID:    apiResp.Data.Attributes.ObjectGroupId,
		ObjectGroupTitle: apiResp.Data.Attributes.ObjectGroupTitle,
		CreatedAt:        timeValue(apiResp.Data.Attributes.CreatedAt),
		UpdatedAt:        timeValue(apiResp.Data.Attributes.UpdatedAt),
	}

	return piggyBank, nil
}

// ListPiggyBanks retrieves a list of piggy banks with pagination
func (c *FireflyClient) ListPiggyBanks(page, limit int) ([]PiggyBankModel, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.ListPiggyBankWithResponse(ctx, &ListPiggyBankParams{
		Page:  int32Ptr(page),
		Limit: int32Ptr(limit),
	})
	if err != nil {
		return nil, APIErr("Failed to list piggy banks", err)
	}

	// Check response
	if resp.StatusCode() == http.StatusTooManyRequests {
		return nil, RateLimitErr(fmt.Errorf("rate limit exceeded"))
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, APIErr("Failed to list piggy banks", fmt.Errorf("unexpected status: %s", resp.Status()))
	}

	// Convert API response to PiggyBankModel array
	if resp.HTTPResponse == nil || len(resp.Body) == 0 {
		return nil, APIErr("No piggy bank data found", fmt.Errorf("empty response"))
	}

	var apiResp PiggyBankArray
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return nil, APIErr("Failed to parse piggy banks response", err)
	}

	piggyBanks := make([]PiggyBankModel, 0, len(apiResp.Data))
	for _, piggyBankRead := range apiResp.Data {
		piggyBank := PiggyBankModel{
			ID:               piggyBankRead.Id,
			Name:             piggyBankRead.Attributes.Name,
			TargetAmount:     stringValue(piggyBankRead.Attributes.TargetAmount),
			CurrentAmount:    stringValue(piggyBankRead.Attributes.CurrentAmount),
			StartDate:        apiDateToTime(piggyBankRead.Attributes.StartDate),
			TargetDate:       apiDateToTime(piggyBankRead.Attributes.TargetDate),
			Notes:            piggyBankRead.Attributes.Notes,
			Order:            piggyBankRead.Attributes.Order,
			Active:           boolValue(piggyBankRead.Attributes.Active),
			Percentage:       float32Value(piggyBankRead.Attributes.Percentage),
			CurrencyCode:     stringValue(piggyBankRead.Attributes.CurrencyCode),
			CurrencySymbol:   stringValue(piggyBankRead.Attributes.CurrencySymbol),
			LeftToSave:       stringValue(piggyBankRead.Attributes.LeftToSave),
			SavePerMonth:     stringValue(piggyBankRead.Attributes.SavePerMonth),
			ObjectGroupID:    piggyBankRead.Attributes.ObjectGroupId,
			ObjectGroupTitle: piggyBankRead.Attributes.ObjectGroupTitle,
			CreatedAt:        timeValue(piggyBankRead.Attributes.CreatedAt),
			UpdatedAt:        timeValue(piggyBankRead.Attributes.UpdatedAt),
		}
		piggyBanks = append(piggyBanks, piggyBank)
	}

	return piggyBanks, nil
}

// UpdatePiggyBank updates an existing piggy bank
func (c *FireflyClient) UpdatePiggyBank(id string, piggyBank PiggyBankModel) error {
	// Validate piggy bank
	if errs := validatePiggyBank(piggyBank); errs != nil {
		return ValidationErr("PiggyBank", errs)
	}

	ctx := context.Background()

	// Create piggy bank update request
	update := PiggyBankUpdate{
		Name:         stringPtr(piggyBank.Name),
		TargetAmount: &piggyBank.TargetAmount,
		StartDate:    dateToAPIDate(piggyBank.StartDate),
		TargetDate:   dateToAPIDate(piggyBank.TargetDate),
		Notes:        piggyBank.Notes,
		Order:        piggyBank.Order,
		Active:       &piggyBank.Active,
	}

	// Call the API
	resp, err := c.clientAPI.UpdatePiggyBankWithResponse(ctx, id, &UpdatePiggyBankParams{}, update)
	if err != nil {
		return APIErr("Failed to update piggy bank", err)
	}

	// Check response
	if resp.StatusCode() == http.StatusNotFound {
		return NotFoundErr("PiggyBank", fmt.Errorf("piggy bank not found: %s", id))
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return RateLimitErr(fmt.Errorf("rate limit exceeded"))
	}
	if resp.StatusCode() != http.StatusOK {
		return APIErr("Failed to update piggy bank", fmt.Errorf("unexpected status: %s", resp.Status()))
	}

	return nil
}

// DeletePiggyBank deletes a piggy bank
func (c *FireflyClient) DeletePiggyBank(id string) error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.DeletePiggyBankWithResponse(ctx, id, &DeletePiggyBankParams{})
	if err != nil {
		return APIErr("Failed to delete piggy bank", err)
	}

	// Check response
	if resp.StatusCode() == http.StatusNotFound {
		return NotFoundErr("PiggyBank", fmt.Errorf("piggy bank not found: %s", id))
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return RateLimitErr(fmt.Errorf("rate limit exceeded"))
	}
	if resp.StatusCode() != http.StatusNoContent {
		return APIErr("Failed to delete piggy bank", fmt.Errorf("unexpected status: %s", resp.Status()))
	}

	return nil
}

// GetPiggyBankEvents retrieves all events for a piggy bank
func (c *FireflyClient) GetPiggyBankEvents(piggyBankID string) ([]PiggyBankEventModel, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.ListEventByPiggyBankWithResponse(ctx, piggyBankID, &ListEventByPiggyBankParams{})
	if err != nil {
		return nil, APIErr("Failed to get piggy bank events", err)
	}

	// Check response
	if resp.StatusCode() == http.StatusNotFound {
		return nil, NotFoundErr("PiggyBank", fmt.Errorf("piggy bank not found: %s", piggyBankID))
	}
	if resp.StatusCode() == http.StatusTooManyRequests {
		return nil, RateLimitErr(fmt.Errorf("rate limit exceeded"))
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, APIErr("Failed to get piggy bank events", fmt.Errorf("unexpected status: %s", resp.Status()))
	}

	// Convert API response to PiggyBankEventModel array
	if resp.HTTPResponse == nil || len(resp.Body) == 0 {
		return nil, APIErr("No piggy bank event data found", fmt.Errorf("empty response"))
	}

	var apiResp PiggyBankEventArray
	if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
		return nil, APIErr("Failed to parse piggy bank events response", err)
	}

	events := make([]PiggyBankEventModel, 0, len(apiResp.Data))
	for _, eventRead := range apiResp.Data {
		event := PiggyBankEventModel{
			ID:                    eventRead.Id,
			PiggyBankID:          piggyBankID,
			TransactionJournalID: stringValue(eventRead.Attributes.TransactionJournalId),
			Amount:               stringValue(eventRead.Attributes.Amount),
			CurrencyCode:         stringValue(eventRead.Attributes.CurrencyCode),
			CurrencySymbol:       stringValue(eventRead.Attributes.CurrencySymbol),
			CreatedAt:            timeValue(eventRead.Attributes.CreatedAt),
			UpdatedAt:            timeValue(eventRead.Attributes.UpdatedAt),
		}
		events = append(events, event)
	}

	return events, nil
}

// validatePiggyBank validates a piggy bank model
func validatePiggyBank(piggyBank PiggyBankModel) []error {
	var errs []error

	if piggyBank.Name == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}

	if piggyBank.TargetAmount == "" {
		errs = append(errs, fmt.Errorf("target amount is required"))
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

// ValidationErr creates a validation error for piggy banks
func ValidationErr(entity string, errs []error) error {
	return fmt.Errorf("%s validation error: %v", entity, errs)
}

// Helper functions for type conversions
func dateToAPIDate(t *time.Time) *types.Date {
	if t == nil {
		return nil
	}
	date := types.Date{Time: *t}
	return &date
}

func apiDateToTime(d *types.Date) *time.Time {
	if d == nil {
		return nil
	}
	return &d.Time
}

func float32Value(f *float32) float32 {
	if f == nil {
		return 0
	}
	return *f
}

// ExportData exports data from Firefly III in the specified format
func (c *FireflyClient) ExportData(dataType DataType, format ExportFormat) ([]byte, error) {
	ctx := context.Background()

	// Validate format
	if format != ExportFormatCSV {
		return nil, ValidationErr("ExportFormat", []error{fmt.Errorf("unsupported format: %s", format)})
	}

	// Build the export endpoint based on data type
	endpoint := fmt.Sprintf("/v1/data/export/%s", dataType)
	
	// Make the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("format", string(format))
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/octet-stream")

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to export data: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	switch resp.StatusCode {
	case http.StatusOK:
		return nil, nil // TODO: Read response body
	case http.StatusNotFound:
		return nil, fmt.Errorf("data export not found: %s", dataType)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to export data", resp.StatusCode)
	}
}

// DestroyData permanently deletes data of the specified type
func (c *FireflyClient) DestroyData(dataType DataType) error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.DestroyData(ctx, &DestroyDataParams{
		Objects: DataDestroyObject(dataType),
	})
	if err != nil {
		return fmt.Errorf("failed to destroy data: %w", err)
	}

	// Check response
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("data type not found: %s", dataType)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to destroy data", resp.StatusCode)
	}
}

// BulkUpdateTransactions updates multiple transactions based on a query
func (c *FireflyClient) BulkUpdateTransactions(query map[string]interface{}) error {
	ctx := context.Background()

	// Convert query to JSON
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}

	// Call the API
	resp, err := c.clientAPI.BulkUpdateTransactions(ctx, &BulkUpdateTransactionsParams{
		Query: json.RawMessage(queryJSON),
	})
	if err != nil {
		return fmt.Errorf("failed to bulk update transactions: %w", err)
	}

	// Check response
	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("invalid bulk update query")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to bulk update transactions", resp.StatusCode)
	}
}

// PurgeData permanently removes all previously deleted data
func (c *FireflyClient) PurgeData() error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.PurgeData(ctx, &PurgeDataParams{})
	if err != nil {
		return fmt.Errorf("failed to purge data: %w", err)
	}

	// Check response
	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to purge data", resp.StatusCode)
	}
}

// CreateTag creates a new tag
func (c *FireflyClient) CreateTag(tag TagModelStore) error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.StoreTagWithResponse(ctx, &StoreTagParams{}, tag)
	if err != nil {
		return fmt.Errorf("failed to create tag: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusConflict:
		return fmt.Errorf("tag already exists")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to create tag", resp.StatusCode())
	}
}

// GetTag retrieves a single tag by ID
func (c *FireflyClient) GetTag(id string) (*TagRead, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.GetTagWithResponse(ctx, id, &GetTagParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get tag: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK:
		if resp.Body == nil {
			return nil, fmt.Errorf("empty response")
		}
		var apiResp TagSingle
		if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse tag response: %w", err)
		}
		return &apiResp.Data, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("tag not found: %s", id)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to get tag", resp.StatusCode())
	}
}

// ListTags retrieves a list of tags with pagination
func (c *FireflyClient) ListTags(page, limit int) ([]TagRead, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.ListTagWithResponse(ctx, &ListTagParams{
		Page:  int32Ptr(page),
		Limit: int32Ptr(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK:
		if resp.Body == nil {
			return nil, fmt.Errorf("empty response")
		}
		var apiResp TagArray
		if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse tags response: %w", err)
		}
		return apiResp.Data, nil
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to list tags", resp.StatusCode())
	}
}

// UpdateTag updates an existing tag
func (c *FireflyClient) UpdateTag(id string, tag TagModelUpdate) error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.UpdateTagWithResponse(ctx, id, &UpdateTagParams{}, tag)
	if err != nil {
		return fmt.Errorf("failed to update tag: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("tag not found: %s", id)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to update tag", resp.StatusCode())
	}
}

// DeleteTag deletes a tag
func (c *FireflyClient) DeleteTag(id string) error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.DeleteTagWithResponse(ctx, id, &DeleteTagParams{})
	if err != nil {
		return fmt.Errorf("failed to delete tag: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("tag not found: %s", id)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to delete tag", resp.StatusCode())
	}
}

// ChartType represents the type of chart to generate
type ChartType string

const (
	ChartTypeDefault ChartType = "default"
	ChartTypeBudget  ChartType = "budget"
	ChartTypeReport  ChartType = "report"
)

// ChartPeriod represents the time period for chart data
type ChartPeriod string

const (
	ChartPeriodDaily   ChartPeriod = "1D"
	ChartPeriodWeekly  ChartPeriod = "1W"
	ChartPeriodMonthly ChartPeriod = "1M"
	ChartPeriodYearly  ChartPeriod = "1Y"
)

// GenerateChart generates a chart of the specified type and period
func (c *FireflyClient) GenerateChart(chartType ChartType, period ChartPeriod, start, end time.Time) ([]byte, error) {
	ctx := context.Background()

	// Build the request manually since charts are not in the OpenAPI spec
	endpoint := fmt.Sprintf("/api/v1/chart/%s", chartType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("period", string(period))
	q.Add("start", start.Format("2006-01-02"))
	q.Add("end", end.Format("2006-01-02"))
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "image/png")

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate chart: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(resp.Body)
	case http.StatusNotFound:
		return nil, fmt.Errorf("chart type not found: %s", chartType)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to generate chart", resp.StatusCode)
	}
}

// ReportType represents the type of report to generate
type ReportType string

const (
	ReportTypeDefault     ReportType = "default"
	ReportTypeBudget     ReportType = "budget"
	ReportTypeCategory   ReportType = "category"
	ReportTypeTag        ReportType = "tag"
	ReportTypeExpense    ReportType = "expense"
	ReportTypeIncome     ReportType = "income"
)

// GenerateReport generates a report of the specified type
func (c *FireflyClient) GenerateReport(reportType ReportType, start, end time.Time, accounts []string) ([]byte, error) {
	ctx := context.Background()

	// Build the request manually since reports are not in the OpenAPI spec
	endpoint := fmt.Sprintf("/api/v1/report/%s", reportType)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("start", start.Format("2006-01-02"))
	q.Add("end", end.Format("2006-01-02"))
	for _, account := range accounts {
		q.Add("accounts[]", account)
	}
	req.URL.RawQuery = q.Encode()

	// Add headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	// Make the request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate report: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	switch resp.StatusCode {
	case http.StatusOK:
		return io.ReadAll(resp.Body)
	case http.StatusNotFound:
		return nil, fmt.Errorf("report type not found: %s", reportType)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to generate report", resp.StatusCode)
	}
}

// CreateBill creates a new bill
func (c *FireflyClient) CreateBill(bill BillModel) error {
	ctx := context.Background()

	// Create bill request
	request := BillStore{
		Name:       bill.Name,
		AmountMin:  bill.AmountMin,
		AmountMax:  bill.AmountMax,
		Date:       bill.Date,
		EndDate:    bill.EndDate,
		Active:     bill.Active,
		CurrencyCode: bill.CurrencyCode,
		CurrencyId:   bill.CurrencyID,
		ExtensionDate: bill.ExtensionDate,
	}

	// Call the API
	resp, err := c.clientAPI.StoreBillWithResponse(ctx, &StoreBillParams{}, request)
	if err != nil {
		return fmt.Errorf("failed to create bill: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK, http.StatusCreated:
		return nil
	case http.StatusConflict:
		return fmt.Errorf("bill already exists")
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to create bill", resp.StatusCode())
	}
}

// GetBill retrieves a single bill by ID
func (c *FireflyClient) GetBill(id string) (*BillModel, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.GetBillWithResponse(ctx, id, &GetBillParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to get bill: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK:
		if resp.Body == nil {
			return nil, fmt.Errorf("empty response")
		}
		var apiResp BillSingle
		if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse bill response: %w", err)
		}

		// Convert API response to BillModel
		bill := &BillModel{
			ID:                   apiResp.Data.Id,
			Name:                 apiResp.Data.Attributes.Name,
			AmountMin:            apiResp.Data.Attributes.AmountMin,
			AmountMax:            apiResp.Data.Attributes.AmountMax,
			Date:                 apiResp.Data.Attributes.Date,
			EndDate:             apiResp.Data.Attributes.EndDate,
			ExtensionDate:       apiResp.Data.Attributes.ExtensionDate,
			CurrencyCode:        apiResp.Data.Attributes.CurrencyCode,
			CurrencyID:          apiResp.Data.Attributes.CurrencyId,
			CurrencySymbol:      apiResp.Data.Attributes.CurrencySymbol,
			CurrencyDecimalPlaces: apiResp.Data.Attributes.CurrencyDecimalPlaces,
			NativeAmountMax:     apiResp.Data.Attributes.NativeAmountMax,
			Active:              apiResp.Data.Attributes.Active,
			CreatedAt:           apiResp.Data.Attributes.CreatedAt,
			UpdatedAt:           apiResp.Data.Attributes.UpdatedAt,
		}

		return bill, nil
	case http.StatusNotFound:
		return nil, fmt.Errorf("bill not found: %s", id)
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to get bill", resp.StatusCode())
	}
}

// ListBills retrieves a list of bills with pagination
func (c *FireflyClient) ListBills(page, limit int) ([]BillModel, error) {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.ListBillWithResponse(ctx, &ListBillParams{
		Page:  int32Ptr(page),
		Limit: int32Ptr(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list bills: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK:
		if resp.Body == nil {
			return nil, fmt.Errorf("empty response")
		}
		var apiResp BillArray
		if err := json.Unmarshal(resp.Body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse bills response: %w", err)
		}

		bills := make([]BillModel, 0, len(apiResp.Data))
		for _, billRead := range apiResp.Data {
			bill := BillModel{
				ID:                   billRead.Id,
				Name:                 billRead.Attributes.Name,
				AmountMin:            billRead.Attributes.AmountMin,
				AmountMax:            billRead.Attributes.AmountMax,
				Date:                 billRead.Attributes.Date,
				EndDate:             billRead.Attributes.EndDate,
				ExtensionDate:       billRead.Attributes.ExtensionDate,
				CurrencyCode:        billRead.Attributes.CurrencyCode,
				CurrencyID:          billRead.Attributes.CurrencyId,
				CurrencySymbol:      billRead.Attributes.CurrencySymbol,
				CurrencyDecimalPlaces: billRead.Attributes.CurrencyDecimalPlaces,
				NativeAmountMax:     billRead.Attributes.NativeAmountMax,
				Active:              billRead.Attributes.Active,
				CreatedAt:           billRead.Attributes.CreatedAt,
				UpdatedAt:           billRead.Attributes.UpdatedAt,
			}
			bills = append(bills, bill)
		}

		return bills, nil
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded")
	default:
		return nil, fmt.Errorf("API error (status %d): failed to list bills", resp.StatusCode())
	}
}

// UpdateBill updates an existing bill
func (c *FireflyClient) UpdateBill(id string, bill BillModel) error {
	ctx := context.Background()

	// Create update request
	update := BillUpdate{
		Name:          bill.Name,
		AmountMin:     stringPtr(bill.AmountMin),
		AmountMax:     stringPtr(bill.AmountMax),
		Date:          &bill.Date,
		EndDate:       bill.EndDate,
		ExtensionDate: bill.ExtensionDate,
		Active:        bill.Active,
		CurrencyCode:  bill.CurrencyCode,
		CurrencyId:    bill.CurrencyID,
		Notes:         bill.Notes,
		ObjectGroupId: bill.ObjectGroupID,
	}

	// Call the API
	resp, err := c.clientAPI.UpdateBillWithResponse(ctx, id, &UpdateBillParams{}, update)
	if err != nil {
		return fmt.Errorf("failed to update bill: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("bill not found: %s", id)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to update bill", resp.StatusCode())
	}
}

// DeleteBill deletes a bill
func (c *FireflyClient) DeleteBill(id string) error {
	ctx := context.Background()

	// Call the API
	resp, err := c.clientAPI.DeleteBillWithResponse(ctx, id, &DeleteBillParams{})
	if err != nil {
		return fmt.Errorf("failed to delete bill: %w", err)
	}

	// Check response
	switch resp.StatusCode() {
	case http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return fmt.Errorf("bill not found: %s", id)
	case http.StatusTooManyRequests:
		return fmt.Errorf("rate limit exceeded")
	default:
		return fmt.Errorf("API error (status %d): failed to delete bill", resp.StatusCode())
	}
}

// Helper function to convert string to pointer
func stringPtr(s string) *string {
	return &s
}

