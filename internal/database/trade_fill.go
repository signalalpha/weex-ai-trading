package database

import (
	"database/sql"
	"fmt"
	"time"

	weexgo "github.com/signalalpha/weex-go"
)

// TradeFill represents a trade fill record in the database
type TradeFill struct {
	ID        int64
	UserID    string
	OrderID   string
	TradeID   string
	Symbol    string
	Side      string
	Price     string
	Size      string
	Fee       string
	FeeCoin   string
	TradeTime int64
	CreatedAt time.Time
}

// SaveTradeFill saves a trade fill to the database
func (db *DB) SaveTradeFill(userID string, fill *weexgo.TradeFill) error {
	query := `
		INSERT INTO trade_fills (
			user_id, order_id, trade_id, symbol, side, 
			price, size, fee, fee_coin, trade_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, order_id, trade_id, trade_time) 
		DO NOTHING
	`

	orderID := fill.GetOrderID()
	_, err := db.Exec(
		query,
		userID,
		orderID,
		fill.GetTradeID(),
		fill.Symbol,
		fill.OrderSide,
		fill.FillValue,
		fill.FillSize,
		fill.FillFee,
		"USDT",
		fill.CreatedTime,
	)

	if err != nil {
		return fmt.Errorf("failed to save trade fill: %w", err)
	}

	return nil
}

// SaveTradeFills saves multiple trade fills in a transaction
func (db *DB) SaveTradeFills(userID string, fills weexgo.TradeFills) (int, error) {
	if len(fills) == 0 {
		return 0, nil
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO trade_fills (
			user_id, order_id, trade_id, symbol, side, 
			price, size, fee, fee_coin, trade_time
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (user_id, order_id, trade_id, trade_time) 
		DO NOTHING
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	savedCount := 0
	for _, fill := range fills {
		orderID := fill.GetOrderID()
		result, err := stmt.Exec(
			userID,
			orderID,
			fill.GetTradeID(),
			fill.Symbol,
			fill.OrderSide,
			fill.FillValue,
			fill.FillSize,
			fill.FillFee,
			"USDT",
			fill.CreatedTime,
		)
		if err != nil {
			continue // Skip this record and continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			savedCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return savedCount, nil
}

// GetLastTradeTime gets the last trade time for a user and symbol
func (db *DB) GetLastTradeTime(userID, symbol string) (int64, error) {
	var lastTradeTime sql.NullInt64
	query := `
		SELECT MAX(trade_time) 
		FROM trade_fills 
		WHERE user_id = $1 AND symbol = $2
	`

	err := db.QueryRow(query, userID, symbol).Scan(&lastTradeTime)
	if err != nil {
		if err == sql.ErrNoRows || !lastTradeTime.Valid {
			return 0, nil // No previous records
		}
		return 0, fmt.Errorf("failed to get last trade time: %w", err)
	}

	if !lastTradeTime.Valid {
		return 0, nil
	}

	return lastTradeTime.Int64, nil
}

// SaveSyncStatus saves sync status
func (db *DB) SaveSyncStatus(userID, symbol string, lastTradeTime int64, recordsCount int, status, errorMsg string) error {
	query := `
		INSERT INTO sync_status (
			user_id, symbol, last_sync_time, last_trade_time, 
			records_count, status, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := db.Exec(
		query,
		userID,
		symbol,
		time.Now(),
		lastTradeTime,
		recordsCount,
		status,
		errorMsg,
	)

	return err
}

// GetTradeFills queries trade fills with filters
func (db *DB) GetTradeFills(userID, symbol string, startTime, endTime int64, limit int) ([]TradeFill, error) {
	query := `
		SELECT id, user_id, order_id, trade_id, symbol, side, 
		       price, size, fee, fee_coin, trade_time, created_at
		FROM trade_fills
		WHERE 1=1
	`

	args := []interface{}{}
	argIndex := 1

	if userID != "" {
		query += fmt.Sprintf(" AND user_id = $%d", argIndex)
		args = append(args, userID)
		argIndex++
	}

	if symbol != "" {
		query += fmt.Sprintf(" AND symbol = $%d", argIndex)
		args = append(args, symbol)
		argIndex++
	}

	if startTime > 0 {
		query += fmt.Sprintf(" AND trade_time >= $%d", argIndex)
		args = append(args, startTime)
		argIndex++
	}

	if endTime > 0 {
		query += fmt.Sprintf(" AND trade_time <= $%d", argIndex)
		args = append(args, endTime)
		argIndex++
	}

	query += " ORDER BY trade_time DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query trade fills: %w", err)
	}
	defer rows.Close()

	var fills []TradeFill
	for rows.Next() {
		var fill TradeFill
		err := rows.Scan(
			&fill.ID,
			&fill.UserID,
			&fill.OrderID,
			&fill.TradeID,
			&fill.Symbol,
			&fill.Side,
			&fill.Price,
			&fill.Size,
			&fill.Fee,
			&fill.FeeCoin,
			&fill.TradeTime,
			&fill.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade fill: %w", err)
		}
		fills = append(fills, fill)
	}

	return fills, rows.Err()
}
