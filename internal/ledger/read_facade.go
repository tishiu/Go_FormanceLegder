package ledger

import (
	"Go_FormanceLegder/internal/api"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerRead interface {
	ForLedger(ledgerID string) LedgerScope
}

type LedgerScope interface {
	ListAccounts(ctx context.Context) ([]AccountResponse, error)
	GetAccount(ctx context.Context, code string) (AccountResponse, error)
	ListTransactions(ctx context.Context, params TransactionListParams) ([]TransactionResponse, api.PaginationResponse, error)
	GetTransaction(ctx context.Context, transactionID string) (TransactionResponse, error)
	ListEvents(ctx context.Context, params EventListParams) ([]EventResponse, api.PaginationResponse, error)
	GetEvent(ctx context.Context, eventID string) (EventResponse, error)
	GetBalanceSummary(ctx context.Context) (BalanceSummaryResponse, error)
	GetAccountBalanceHistory(ctx context.Context, accountCode string) (AccountBalanceHistoryResponse, error)
}

type TransactionListParams struct {
	Limit             int
	ContinuationToken string
	StartTime         string
	EndTime           string
}

type EventListParams struct {
	Limit             int
	ContinuationToken string
	EventType         string
	AggregateID       string
}

type SQLLedgerRead struct {
	DB *pgxpool.Pool
}

type sqlLedgerScope struct {
	db       *pgxpool.Pool
	ledgerID string
}

func NewSQLLedgerRead(db *pgxpool.Pool) *SQLLedgerRead {
	return &SQLLedgerRead{DB: db}
}

func (r *SQLLedgerRead) ForLedger(ledgerID string) LedgerScope {
	return &sqlLedgerScope{
		db:       r.DB,
		ledgerID: ledgerID,
	}
}

func (s *sqlLedgerScope) ListAccounts(ctx context.Context) ([]AccountResponse, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, code, name, type, balance, created_at
		FROM accounts
		WHERE ledger_id = $1
		ORDER BY code
	`, s.ledgerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	accounts := []AccountResponse{}
	for rows.Next() {
		var acc AccountResponse
		if err := rows.Scan(&acc.ID, &acc.Code, &acc.Name, &acc.Type, &acc.Balance, &acc.CreatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, acc)
	}
	return accounts, rows.Err()
}

func (s *sqlLedgerScope) GetAccount(ctx context.Context, code string) (AccountResponse, error) {
	var acc AccountResponse
	err := s.db.QueryRow(ctx, `
		SELECT id, code, name, type, balance, created_at
		FROM accounts
		WHERE ledger_id = $1 AND code = $2
	`, s.ledgerID, code).Scan(&acc.ID, &acc.Code, &acc.Name, &acc.Type, &acc.Balance, &acc.CreatedAt)
	return acc, err
}

func (s *sqlLedgerScope) ListTransactions(ctx context.Context, params TransactionListParams) ([]TransactionResponse, api.PaginationResponse, error) {
	limit := api.ValidateLimit(params.Limit)
	cursor, err := api.DecodeCursor(params.ContinuationToken)
	if err != nil {
		return nil, api.PaginationResponse{}, err
	}

	query := `
		SELECT t.id, t.external_id, t.amount, t.currency, t.occurred_at, t.created_at
		FROM transactions t
		WHERE t.ledger_id = $1
	`
	args := []interface{}{s.ledgerID}
	argCount := 1

	if !cursor.Timestamp.IsZero() {
		argCount++
		query += ` AND (t.created_at, t.id) < ($` + fmt.Sprintf("%d", argCount) + `, $` + fmt.Sprintf("%d", argCount+1) + `)`
		args = append(args, cursor.Timestamp, cursor.ID)
		argCount++
	}
	if params.StartTime != "" {
		argCount++
		query += ` AND t.occurred_at >= $` + fmt.Sprintf("%d", argCount)
		args = append(args, params.StartTime)
	}
	if params.EndTime != "" {
		argCount++
		query += ` AND t.occurred_at <= $` + fmt.Sprintf("%d", argCount)
		args = append(args, params.EndTime)
	}

	query += ` ORDER BY t.created_at DESC, t.id DESC LIMIT $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit+1)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, api.PaginationResponse{}, err
	}
	defer rows.Close()

	type txRow struct {
		txn       TransactionResponse
		createdAt time.Time
	}
	all := []txRow{}

	for rows.Next() {
		var (
			txn        TransactionResponse
			occurredAt time.Time
			createdAt  time.Time
		)
		if err := rows.Scan(&txn.ID, &txn.ExternalID, &txn.Amount, &txn.Currency, &occurredAt, &createdAt); err != nil {
			return nil, api.PaginationResponse{}, err
		}
		txn.OccurredAt = occurredAt.Format(time.RFC3339)
		txn.CreatedAt = createdAt.Format(time.RFC3339)

		postings, err := s.loadPostings(ctx, txn.ID)
		if err != nil {
			return nil, api.PaginationResponse{}, err
		}
		txn.Postings = postings
		all = append(all, txRow{txn: txn, createdAt: createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, api.PaginationResponse{}, err
	}

	hasMore := len(all) > limit
	if hasMore {
		all = all[:limit]
	}

	transactions := make([]TransactionResponse, 0, len(all))
	for _, row := range all {
		transactions = append(transactions, row.txn)
	}

	nextToken := ""
	if hasMore && len(all) > 0 {
		nextCursor := api.Cursor{
			Timestamp: all[len(all)-1].createdAt,
			ID:        all[len(all)-1].txn.ID,
		}
		nextToken, _ = api.EncodeCursor(nextCursor)
	}

	return transactions, api.PaginationResponse{
		HasMore:           hasMore,
		ContinuationToken: nextToken,
		Count:             len(transactions),
	}, nil
}

func (s *sqlLedgerScope) GetTransaction(ctx context.Context, transactionID string) (TransactionResponse, error) {
	var (
		txn       TransactionResponse
		occurred  time.Time
		createdAt time.Time
	)
	err := s.db.QueryRow(ctx, `
		SELECT id, external_id, amount, currency, occurred_at, created_at
		FROM transactions
		WHERE ledger_id = $1 AND id = $2
	`, s.ledgerID, transactionID).Scan(&txn.ID, &txn.ExternalID, &txn.Amount, &txn.Currency, &occurred, &createdAt)
	if err != nil {
		return TransactionResponse{}, err
	}
	txn.OccurredAt = occurred.Format(time.RFC3339)
	txn.CreatedAt = createdAt.Format(time.RFC3339)

	postings, err := s.loadPostings(ctx, txn.ID)
	if err != nil {
		return TransactionResponse{}, err
	}
	txn.Postings = postings
	return txn, nil
}

func (s *sqlLedgerScope) ListEvents(ctx context.Context, params EventListParams) ([]EventResponse, api.PaginationResponse, error) {
	limit := api.ValidateLimit(params.Limit)
	cursor, err := api.DecodeCursor(params.ContinuationToken)
	if err != nil {
		return nil, api.PaginationResponse{}, err
	}

	query := `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, occurred_at, created_at
		FROM events
		WHERE ledger_id = $1
	`
	args := []interface{}{s.ledgerID}
	argCount := 1

	if !cursor.Timestamp.IsZero() {
		argCount++
		query += ` AND (created_at, id) < ($` + fmt.Sprintf("%d", argCount) + `, $` + fmt.Sprintf("%d", argCount+1) + `)`
		args = append(args, cursor.Timestamp, cursor.ID)
		argCount++
	}
	if params.EventType != "" {
		argCount++
		query += ` AND event_type = $` + fmt.Sprintf("%d", argCount)
		args = append(args, params.EventType)
	}
	if params.AggregateID != "" {
		argCount++
		query += ` AND aggregate_id = $` + fmt.Sprintf("%d", argCount)
		args = append(args, params.AggregateID)
	}

	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + fmt.Sprintf("%d", argCount+1)
	args = append(args, limit+1)

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, api.PaginationResponse{}, err
	}
	defer rows.Close()

	type evtRow struct {
		evt       EventResponse
		createdAt time.Time
	}
	all := []evtRow{}

	for rows.Next() {
		var (
			evt        EventResponse
			payloadRaw []byte
			occurredAt time.Time
			createdAt  time.Time
		)
		if err := rows.Scan(&evt.ID, &evt.AggregateType, &evt.AggregateID, &evt.EventType, &payloadRaw, &occurredAt, &createdAt); err != nil {
			return nil, api.PaginationResponse{}, err
		}
		if err := json.Unmarshal(payloadRaw, &evt.Payload); err != nil {
			return nil, api.PaginationResponse{}, err
		}
		evt.OccurredAt = occurredAt.Format(time.RFC3339)
		evt.CreatedAt = createdAt.Format(time.RFC3339)
		all = append(all, evtRow{evt: evt, createdAt: createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, api.PaginationResponse{}, err
	}

	hasMore := len(all) > limit
	if hasMore {
		all = all[:limit]
	}

	events := make([]EventResponse, 0, len(all))
	for _, row := range all {
		events = append(events, row.evt)
	}

	nextToken := ""
	if hasMore && len(all) > 0 {
		nextCursor := api.Cursor{
			Timestamp: all[len(all)-1].createdAt,
			ID:        all[len(all)-1].evt.ID,
		}
		nextToken, _ = api.EncodeCursor(nextCursor)
	}

	return events, api.PaginationResponse{
		HasMore:           hasMore,
		ContinuationToken: nextToken,
		Count:             len(events),
	}, nil
}

func (s *sqlLedgerScope) GetEvent(ctx context.Context, eventID string) (EventResponse, error) {
	var (
		evt        EventResponse
		payloadRaw []byte
		occurredAt time.Time
		createdAt  time.Time
	)
	err := s.db.QueryRow(ctx, `
		SELECT id, aggregate_type, aggregate_id, event_type, payload, occurred_at, created_at
		FROM events
		WHERE ledger_id = $1 AND id = $2
	`, s.ledgerID, eventID).Scan(&evt.ID, &evt.AggregateType, &evt.AggregateID, &evt.EventType, &payloadRaw, &occurredAt, &createdAt)
	if err != nil {
		return EventResponse{}, err
	}
	if err := json.Unmarshal(payloadRaw, &evt.Payload); err != nil {
		return EventResponse{}, err
	}
	evt.OccurredAt = occurredAt.Format(time.RFC3339)
	evt.CreatedAt = createdAt.Format(time.RFC3339)
	return evt, nil
}

func (s *sqlLedgerScope) GetBalanceSummary(ctx context.Context) (BalanceSummaryResponse, error) {
	rows, err := s.db.Query(ctx, `
		SELECT type, COALESCE(SUM(balance)::text, '0') as total
		FROM accounts
		WHERE ledger_id = $1
		GROUP BY type
	`, s.ledgerID)
	if err != nil {
		return BalanceSummaryResponse{}, err
	}
	defer rows.Close()

	summary := BalanceSummaryResponse{
		TotalAssets:      "0",
		TotalLiabilities: "0",
		TotalEquity:      "0",
		TotalRevenue:     "0",
		TotalExpenses:    "0",
		ByType:           make(map[string]string),
	}

	for rows.Next() {
		var accountType, total string
		if err := rows.Scan(&accountType, &total); err != nil {
			return BalanceSummaryResponse{}, err
		}
		summary.ByType[accountType] = total
		switch accountType {
		case "asset":
			summary.TotalAssets = total
		case "liability":
			summary.TotalLiabilities = total
		case "equity":
			summary.TotalEquity = total
		case "revenue":
			summary.TotalRevenue = total
		case "expense":
			summary.TotalExpenses = total
		}
	}
	return summary, rows.Err()
}

func (s *sqlLedgerScope) GetAccountBalanceHistory(ctx context.Context, accountCode string) (AccountBalanceHistoryResponse, error) {
	var accountID string
	if err := s.db.QueryRow(ctx, `
		SELECT id FROM accounts WHERE ledger_id = $1 AND code = $2
	`, s.ledgerID, accountCode).Scan(&accountID); err != nil {
		return AccountBalanceHistoryResponse{}, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT 
			DATE(t.occurred_at)::text as date,
			COALESCE(SUM(CASE WHEN p.direction = 'debit' THEN p.amount ELSE -p.amount END)::text, '0') as net_change
		FROM postings p
		JOIN transactions t ON t.id = p.transaction_id
		WHERE p.account_id = $1
		GROUP BY DATE(t.occurred_at)
		ORDER BY date ASC
	`, accountID)
	if err != nil {
		return AccountBalanceHistoryResponse{}, err
	}
	defer rows.Close()

	history := []BalanceHistoryPoint{}
	running := new(big.Rat).SetInt64(0)

	for rows.Next() {
		var (
			date      string
			netChange string
		)
		if err := rows.Scan(&date, &netChange); err != nil {
			return AccountBalanceHistoryResponse{}, err
		}
		change := new(big.Rat)
		if _, ok := change.SetString(netChange); !ok {
			return AccountBalanceHistoryResponse{}, fmt.Errorf("invalid net change amount: %s", netChange)
		}
		running = new(big.Rat).Add(running, change)
		history = append(history, BalanceHistoryPoint{
			Date:    date,
			Balance: running.FloatString(2),
		})
	}
	if err := rows.Err(); err != nil {
		return AccountBalanceHistoryResponse{}, err
	}

	return AccountBalanceHistoryResponse{
		AccountCode: accountCode,
		History:     history,
	}, nil
}

func (s *sqlLedgerScope) loadPostings(ctx context.Context, transactionID string) ([]PostingDetail, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.id, a.code, a.name, p.direction, p.amount
		FROM postings p
		JOIN accounts a ON a.id = p.account_id
		WHERE p.ledger_id = $1 AND p.transaction_id = $2
		ORDER BY p.created_at
	`, s.ledgerID, transactionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	postings := []PostingDetail{}
	for rows.Next() {
		var p PostingDetail
		if err := rows.Scan(&p.ID, &p.AccountCode, &p.AccountName, &p.Direction, &p.Amount); err != nil {
			return nil, err
		}
		postings = append(postings, p)
	}
	return postings, rows.Err()
}
