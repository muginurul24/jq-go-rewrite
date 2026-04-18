package handlers

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"

	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/auth"
	"github.com/mugiew/justqiuv2-rewrite/apps/api/internal/modules/transactions"
)

type BackofficeTransactionsHandler struct {
	service backofficeTransactionsService
}

type backofficeTransactionsService interface {
	ListForBackoffice(ctx context.Context, user auth.PublicUser, input transactions.AdminListInput) (*transactions.AdminListResult, error)
	FindDetailForBackoffice(ctx context.Context, user auth.PublicUser, transactionID int64) (*transactions.AdminTransactionDetail, error)
	ExportForBackoffice(ctx context.Context, user auth.PublicUser, input transactions.AdminListInput) ([]transactions.AdminTransactionRecord, error)
}

type backofficeTransactionRecordResponse struct {
	ID             int64   `json:"id"`
	TokoID         int64   `json:"tokoId"`
	TokoName       string  `json:"tokoName"`
	OwnerUsername  string  `json:"ownerUsername"`
	Player         *string `json:"player,omitempty"`
	ExternalPlayer *string `json:"externalPlayer,omitempty"`
	Category       string  `json:"category"`
	CategoryLabel  string  `json:"categoryLabel"`
	Type           string  `json:"type"`
	TypeLabel      string  `json:"typeLabel"`
	Status         string  `json:"status"`
	StatusLabel    string  `json:"statusLabel"`
	Amount         int64   `json:"amount"`
	Code           *string `json:"code,omitempty"`
	NoteSummary    *string `json:"noteSummary,omitempty"`
	CreatedAt      string  `json:"createdAt"`
	UpdatedAt      string  `json:"updatedAt"`
	CanEdit        bool    `json:"canEdit"`
	CanDelete      bool    `json:"canDelete"`
}

type backofficeTransactionDetailResponse struct {
	backofficeTransactionRecordResponse
	NotePayload string `json:"notePayload"`
}

func NewBackofficeTransactionsHandler(service backofficeTransactionsService) *BackofficeTransactionsHandler {
	return &BackofficeTransactionsHandler{service: service}
}

func (h *BackofficeTransactionsHandler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	input, err := parseBackofficeTransactionListInput(r)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		return
	}

	result, err := h.service.ListForBackoffice(r.Context(), user, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load transactions"})
		return
	}

	rows := make([]backofficeTransactionRecordResponse, 0, len(result.Items))
	for _, record := range result.Items {
		rows = append(rows, presentBackofficeTransaction(record, user))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": rows,
		"meta": map[string]any{
			"page":       result.Page,
			"perPage":    result.PerPage,
			"total":      result.Total,
			"totalPages": result.TotalPages,
		},
		"summary": map[string]any{
			"totalAmount": result.TotalAmount,
		},
		"filters": map[string]any{
			"tokos": result.TokoOptions,
		},
	})
}

func (h *BackofficeTransactionsHandler) Detail(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	transactionID, err := strconv.ParseInt(chi.URLParam(r, "transactionID"), 10, 64)
	if err != nil || transactionID <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "Invalid transaction id"})
		return
	}

	detail, err := h.service.FindDetailForBackoffice(r.Context(), user, transactionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "Transaction not found"})
			return
		}

		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to load transaction detail"})
		return
	}

	record := presentBackofficeTransaction(detail.Record, user)
	writeJSON(w, http.StatusOK, map[string]any{
		"data": backofficeTransactionDetailResponse{
			backofficeTransactionRecordResponse: record,
			NotePayload:                         detail.NotePayload,
		},
	})
}

func (h *BackofficeTransactionsHandler) Export(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.CurrentUser(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	input, err := parseBackofficeTransactionListInput(r)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": err.Error()})
		return
	}

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "csv"
	}
	if format != "csv" && format != "xlsx" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": "Invalid export format"})
		return
	}

	records, err := h.service.ExportForBackoffice(r.Context(), user, input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to export transactions"})
		return
	}

	filename := fmt.Sprintf("transactions-%s.%s", time.Now().UTC().Format("20060102-150405"), format)
	switch format {
	case "xlsx":
		payload, err := renderTransactionsXLSX(records)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to render transaction export"})
			return
		}
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	default:
		payload, err := renderTransactionsCSV(records)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to render transaction export"})
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}
}

func parseBackofficeTransactionListInput(r *http.Request) (transactions.AdminListInput, error) {
	query := r.URL.Query()

	page, err := parsePositiveInt(query.Get("page"), 1)
	if err != nil {
		return transactions.AdminListInput{}, err
	}

	perPage, err := parsePositiveInt(query.Get("per_page"), 25)
	if err != nil {
		return transactions.AdminListInput{}, err
	}

	tokoIDs, err := parseInt64Slice(query.Get("toko_ids"))
	if err != nil {
		return transactions.AdminListInput{}, err
	}

	amountMin, err := parseOptionalInt64(query.Get("amount_min"))
	if err != nil {
		return transactions.AdminListInput{}, err
	}

	amountMax, err := parseOptionalInt64(query.Get("amount_max"))
	if err != nil {
		return transactions.AdminListInput{}, err
	}

	dateFrom := optionalTrimmed(query.Get("date_from"))
	dateUntil := optionalTrimmed(query.Get("date_until"))

	return transactions.AdminListInput{
		Search:        strings.TrimSpace(query.Get("search")),
		Categories:    parseCSV(query.Get("categories")),
		Types:         parseCSV(query.Get("types")),
		Statuses:      parseCSV(query.Get("statuses")),
		TokoIDs:       tokoIDs,
		DateFrom:      dateFrom,
		DateUntil:     dateUntil,
		AmountMin:     amountMin,
		AmountMax:     amountMax,
		Page:          page,
		PerPage:       perPage,
		SortBy:        strings.TrimSpace(query.Get("sort_by")),
		SortDirection: strings.TrimSpace(query.Get("sort_direction")),
	}, nil
}

func parsePositiveInt(raw string, fallback int) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return fallback, nil
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || parsed <= 0 {
		return 0, errors.New("Invalid pagination parameter")
	}

	return parsed, nil
}

func parseOptionalInt64(raw string) (*int64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return nil, errors.New("Invalid amount filter")
	}

	return &parsed, nil
}

func parseInt64Slice(raw string) ([]int64, error) {
	values := parseCSV(raw)
	if len(values) == 0 {
		return nil, nil
	}

	items := make([]int64, 0, len(values))
	for _, value := range values {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return nil, errors.New("Invalid toko filter")
		}
		items = append(items, parsed)
	}

	return items, nil
}

func parseCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}

	return result
}

func optionalTrimmed(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	return &trimmed
}

func presentBackofficeTransaction(record transactions.AdminTransactionRecord, user auth.PublicUser) backofficeTransactionRecordResponse {
	canMutate := user.Role == "dev"
	noteSummary := record.Note
	if summary := transactionsSummary(record.Note); summary != nil {
		noteSummary = summary
	}

	return backofficeTransactionRecordResponse{
		ID:             record.ID,
		TokoID:         record.TokoID,
		TokoName:       record.TokoName,
		OwnerUsername:  record.OwnerUsername,
		Player:         record.Player,
		ExternalPlayer: record.ExternalPlayer,
		Category:       record.Category,
		CategoryLabel:  formatTransactionCategory(record.Category),
		Type:           record.Type,
		TypeLabel:      headlineTransaction(record.Type),
		Status:         record.Status,
		StatusLabel:    headlineTransaction(record.Status),
		Amount:         record.Amount,
		Code:           record.Code,
		NoteSummary:    noteSummary,
		CreatedAt:      record.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      record.UpdatedAt.Format(time.RFC3339),
		CanEdit:        canMutate,
		CanDelete:      canMutate,
	}
}

func formatTransactionCategory(value string) string {
	switch value {
	case "qris":
		return "QRIS"
	case "nexusggr":
		return "NexusGGR"
	default:
		return headlineTransaction(value)
	}
}

func headlineTransaction(value string) string {
	if value == "" {
		return ""
	}

	parts := strings.Split(strings.ReplaceAll(value, "_", " "), " ")
	for index, part := range parts {
		if part == "" {
			continue
		}
		parts[index] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}

	return strings.Join(parts, " ")
}

func transactionsSummary(note *string) *string {
	if note == nil || strings.TrimSpace(*note) == "" {
		return nil
	}

	trimmed := strings.TrimSpace(*note)
	return &trimmed
}

func renderTransactionsCSV(records []transactions.AdminTransactionRecord) ([]byte, error) {
	buffer := &bytes.Buffer{}
	writer := csv.NewWriter(buffer)

	rows := buildTransactionExportRows(records)
	for _, row := range rows {
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func renderTransactionsXLSX(records []transactions.AdminTransactionRecord) ([]byte, error) {
	workbook := excelize.NewFile()
	defer func() {
		_ = workbook.Close()
	}()

	const sheetName = "Transactions"
	workbook.SetSheetName(workbook.GetSheetName(0), sheetName)

	rows := buildTransactionExportRows(records)
	for rowIndex, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, rowIndex+1)
		if err != nil {
			return nil, err
		}
		values := make([]any, 0, len(row))
		for _, value := range row {
			values = append(values, value)
		}
		if err := workbook.SetSheetRow(sheetName, cell, &values); err != nil {
			return nil, err
		}
	}

	if err := workbook.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	}); err != nil {
		return nil, err
	}

	buffer, err := workbook.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func buildTransactionExportRows(records []transactions.AdminTransactionRecord) [][]string {
	rows := [][]string{{
		"ID",
		"Toko",
		"Owner",
		"Player",
		"External Player",
		"Category",
		"Type",
		"Status",
		"Amount",
		"Reference",
		"Created At",
		"Updated At",
		"Note",
	}}

	for _, record := range records {
		rows = append(rows, []string{
			strconv.FormatInt(record.ID, 10),
			record.TokoName,
			record.OwnerUsername,
			safeString(record.Player),
			safeString(record.ExternalPlayer),
			formatTransactionCategory(record.Category),
			headlineTransaction(record.Type),
			headlineTransaction(record.Status),
			strconv.FormatInt(record.Amount, 10),
			safeString(record.Code),
			record.CreatedAt.UTC().Format(time.RFC3339),
			record.UpdatedAt.UTC().Format(time.RFC3339),
			exportNote(record.Note),
		})
	}

	return rows
}

func safeString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func exportNote(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
