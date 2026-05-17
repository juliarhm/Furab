// Package repository provides data access layer for pricing-service.
package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"furab-backend/services/pricing-service/internal/model"
)

var (
	ErrPriceRuleNotFound = errors.New("price rule not found")
)

// PriceRepository defines the interface for pricing-service pricing rule access.
type PriceRepository interface {
	GetPricingRules(ctx context.Context) ([]model.PriceRule, error)
	GetPricingRuleByType(ctx context.Context, ruleType string) (*model.PriceRule, error)
}

type inMemoryPriceRepository struct {
	rules []model.PriceRule
}

// NewInMemoryPriceRepository creates a pricing repository with fixed rules.
func NewInMemoryPriceRepository() PriceRepository {
	return &inMemoryPriceRepository{
		rules: []model.PriceRule{
			{RuleID: "delivery-per-km", Type: "delivery", Value: 5000, Description: "Delivery fee per kilometer"},
			{RuleID: "service-percent", Type: "service", Value: 0.05, Description: "Service fee percentage"},
			{RuleID: "tax-percent", Type: "tax", Value: 0.0, Description: "Tax percentage"},
		},
	}
}

func (r *inMemoryPriceRepository) GetPricingRules(ctx context.Context) ([]model.PriceRule, error) {
	return r.rules, nil
}

func (r *inMemoryPriceRepository) GetPricingRuleByType(ctx context.Context, ruleType string) (*model.PriceRule, error) {
	for _, rule := range r.rules {
		if rule.Type == ruleType {
			copy := rule
			return &copy, nil
		}
	}

	return nil, ErrPriceRuleNotFound
}

type postgresPriceRepository struct {
	db *sql.DB
}

// NewPostgresPriceRepository creates a new PostgreSQL-based repository.
func NewPostgresPriceRepository(db *sql.DB) PriceRepository {
	return &postgresPriceRepository{db: db}
}

func (r *postgresPriceRepository) GetPricingRules(ctx context.Context) ([]model.PriceRule, error) {
	query := `
		SELECT rule_id, type, value, description
		FROM pricing_rules
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pricing rules: %w", err)
	}
	defer rows.Close()

	var rules []model.PriceRule
	for rows.Next() {
		var rule model.PriceRule
		if err := rows.Scan(&rule.RuleID, &rule.Type, &rule.Value, &rule.Description); err != nil {
			return nil, fmt.Errorf("failed to scan pricing rule: %w", err)
		}
		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}
	return rules, nil
}

func (r *postgresPriceRepository) GetPricingRuleByType(ctx context.Context, ruleType string) (*model.PriceRule, error) {
	query := `
		SELECT rule_id, type, value, description
		FROM pricing_rules
		WHERE type = $1
		LIMIT 1
	`

	var rule model.PriceRule
	err := r.db.QueryRowContext(ctx, query, ruleType).Scan(
		&rule.RuleID,
		&rule.Type,
		&rule.Value,
		&rule.Description,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrPriceRuleNotFound
		}
		return nil, fmt.Errorf("failed to get pricing rule by type: %w", err)
	}
	return &rule, nil
}
