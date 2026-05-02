package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/yibaiba/dramora/internal/domain"
	"github.com/yibaiba/dramora/internal/service"
)

type walletDTO struct {
	OrganizationID string `json:"organization_id"`
	Balance        int64  `json:"balance"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type walletTransactionDTO struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Kind           string `json:"kind"`
	Direction      int    `json:"direction"`
	Amount         int64  `json:"amount"`
	Reason         string `json:"reason,omitempty"`
	RefType        string `json:"ref_type,omitempty"`
	RefID          string `json:"ref_id,omitempty"`
	BalanceAfter   int64  `json:"balance_after"`
	ActorUserID    string `json:"actor_user_id,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func walletToDTO(w domain.Wallet) walletDTO {
	dto := walletDTO{OrganizationID: w.OrganizationID, Balance: w.Balance}
	if !w.UpdatedAt.IsZero() {
		dto.UpdatedAt = w.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z")
	}
	return dto
}

func walletTxToDTO(tx domain.WalletTransaction) walletTransactionDTO {
	return walletTransactionDTO{
		ID:             tx.ID,
		OrganizationID: tx.OrganizationID,
		Kind:           string(tx.Kind),
		Direction:      tx.Direction,
		Amount:         tx.Amount,
		Reason:         tx.Reason,
		RefType:        tx.RefType,
		RefID:          tx.RefID,
		BalanceAfter:   tx.BalanceAfter,
		ActorUserID:    tx.ActorUserID,
		CreatedAt:      tx.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}

func (a *api) getWallet(w http.ResponseWriter, r *http.Request) {
	if a.walletService == nil {
		writeError(w, http.StatusServiceUnavailable, "wallet_unavailable", "wallet service not configured")
		return
	}
	snap, err := a.walletService.GetWallet(r.Context())
	if err != nil {
		writeWalletError(w, err)
		return
	}
	txs := make([]walletTransactionDTO, 0, len(snap.RecentTransactions))
	for _, tx := range snap.RecentTransactions {
		txs = append(txs, walletTxToDTO(tx))
	}
	writeJSON(w, http.StatusOK, Envelope{
		"wallet":              walletToDTO(snap.Wallet),
		"recent_transactions": txs,
	})
}

func (a *api) listWalletTransactions(w http.ResponseWriter, r *http.Request) {
	if a.walletService == nil {
		writeError(w, http.StatusServiceUnavailable, "wallet_unavailable", "wallet service not configured")
		return
	}
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	var kinds []string
	if k := strings.TrimSpace(q.Get("kind")); k != "" {
		for _, item := range strings.Split(k, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				kinds = append(kinds, item)
			}
		}
	}
	page, err := a.walletService.ListTransactions(r.Context(), limit, offset, kinds)
	if err != nil {
		writeWalletError(w, err)
		return
	}
	out := make([]walletTransactionDTO, 0, len(page.Transactions))
	for _, tx := range page.Transactions {
		out = append(out, walletTxToDTO(tx))
	}
	writeJSON(w, http.StatusOK, Envelope{
		"transactions": out,
		"has_more":     page.HasMore,
	})
}

type walletMutationRequest struct {
	Amount  int64  `json:"amount"`
	Reason  string `json:"reason"`
	RefType string `json:"ref_type"`
	RefID   string `json:"ref_id"`
}

type operationCostDTO struct {
	Operation string `json:"operation"`
	Cost      int64  `json:"cost"`
}

type previewCostRequest struct {
	OperationType string `json:"operation_type"`
}

type previewCostResponse struct {
	OperationType string `json:"operation_type"`
	Cost          int64  `json:"cost"`
}

func (a *api) creditWallet(w http.ResponseWriter, r *http.Request) {
	a.applyWalletMutation(w, r, true)
}

func (a *api) debitWallet(w http.ResponseWriter, r *http.Request) {
	a.applyWalletMutation(w, r, false)
}

func (a *api) getOperationCosts(w http.ResponseWriter, r *http.Request) {
	if a.walletService == nil {
		writeError(w, http.StatusServiceUnavailable, "wallet_unavailable", "wallet service not configured")
		return
	}
	costs := make([]operationCostDTO, 0, len(domain.OperationCosts))
	for op, cost := range domain.OperationCosts {
		costs = append(costs, operationCostDTO{
			Operation: string(op),
			Cost:      cost,
		})
	}
	writeJSON(w, http.StatusOK, Envelope{"costs": costs})
}

func (a *api) previewWalletCost(w http.ResponseWriter, r *http.Request) {
	if a.walletService == nil {
		writeError(w, http.StatusServiceUnavailable, "wallet_unavailable", "wallet service not configured")
		return
	}
	var req previewCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	opType := domain.OperationType(req.OperationType)
	cost, err := domain.GetOperationCost(opType)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"preview": previewCostResponse{
		OperationType: string(opType),
		Cost:          cost,
	}})
}

func (a *api) applyWalletMutation(w http.ResponseWriter, r *http.Request, credit bool) {
	if a.walletService == nil {
		writeError(w, http.StatusServiceUnavailable, "wallet_unavailable", "wallet service not configured")
		return
	}
	var req walletMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "amount must be positive")
		return
	}
	params := service.CreditParams{
		Amount:  req.Amount,
		Reason:  req.Reason,
		RefType: req.RefType,
		RefID:   req.RefID,
	}
	var (
		tx  domain.WalletTransaction
		err error
	)
	if credit {
		tx, err = a.walletService.Credit(r.Context(), params)
	} else {
		tx, err = a.walletService.Debit(r.Context(), params)
	}
	if err != nil {
		writeWalletError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{"transaction": walletTxToDTO(tx)})
}

func writeWalletError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required")
	case errors.Is(err, service.ErrInsufficientBalance):
		writeError(w, http.StatusUnprocessableEntity, "insufficient_balance", "wallet balance is insufficient")
	default:
		if err != nil && strings.Contains(err.Error(), "invalid kind") {
			writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		writeServiceError(w, err)
	}
}
