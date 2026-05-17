-- Migration: 001_create_payments.sql
-- Creates the payments, payment_logs, and payment_methods tables for the payment-service.
--
-- Run this migration manually or via a migration tool:
--   psql -U furab -d payment_service -f migrations/001_create_payments.sql

-- Create payment_methods table
CREATE TABLE IF NOT EXISTS payment_methods (
    id                 VARCHAR(36) PRIMARY KEY,
    method_name        VARCHAR(100) NOT NULL,
    provider           VARCHAR(50) NOT NULL,
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create payments table (Main transaction table)
CREATE TABLE IF NOT EXISTS payments (
    id                      VARCHAR(36) PRIMARY KEY,
    order_id                VARCHAR(36) NOT NULL,
    user_id                 VARCHAR(36) NOT NULL,
    amount                  DOUBLE PRECISION NOT NULL,
    final_amount            DOUBLE PRECISION NOT NULL,
    method_id               VARCHAR(36) NOT NULL REFERENCES payment_methods(id),
    payment_detail          TEXT NOT NULL,
    payment_status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    transaction_reference   VARCHAR(100) NOT NULL,
    idempotency_key         VARCHAR(100) UNIQUE,
    transaction_time        TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Create payment_logs table (Audit Trail)
CREATE TABLE IF NOT EXISTS payment_logs (
    id              SERIAL PRIMARY KEY,
    payment_id      VARCHAR(36) NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    status          VARCHAR(20) NOT NULL,
    timestamp       TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(payment_status);
CREATE INDEX IF NOT EXISTS idx_payments_idempotency_key ON payments(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_payments_transaction_reference ON payments(transaction_reference);
CREATE INDEX IF NOT EXISTS idx_payments_created_at ON payments(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_payment_logs_payment_id ON payment_logs(payment_id);
CREATE INDEX IF NOT EXISTS idx_payment_logs_status ON payment_logs(status);
CREATE INDEX IF NOT EXISTS idx_payment_logs_timestamp ON payment_logs(timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_payment_methods_provider ON payment_methods(provider);

-- Add status check constraint for payments
ALTER TABLE payments 
ADD CONSTRAINT chk_payment_status 
CHECK (payment_status IN ('pending', 'authorized', 'captured', 'refunded', 'failed', 'cancelled'));

-- Add status check constraint for payment_logs
ALTER TABLE payment_logs 
ADD CONSTRAINT chk_payment_log_status 
CHECK (status IN ('pending', 'authorized', 'captured', 'refunded', 'failed', 'cancelled'));

-- Add amount validation constraint
ALTER TABLE payments 
ADD CONSTRAINT chk_payment_amounts 
CHECK (amount > 0 AND final_amount > 0 AND final_amount <= amount);

-- Comments on tables
COMMENT ON TABLE payments IS 'Stores all payment transactions with two-phase payment flow (authorize & capture)';
COMMENT ON TABLE payment_logs IS 'Audit trail for all payment status changes';
COMMENT ON TABLE payment_methods IS 'Available payment methods and providers';

COMMENT ON COLUMN payments.id IS 'UUID primary key for payment transaction';
COMMENT ON COLUMN payments.order_id IS 'Reference to order in food-order-service or ride-order-service';
COMMENT ON COLUMN payments.user_id IS 'User initiating the payment';
COMMENT ON COLUMN payments.amount IS 'Original amount in IDR (Rupiah)';
COMMENT ON COLUMN payments.final_amount IS 'Amount after discounts from promo-service';
COMMENT ON COLUMN payments.method_id IS 'Payment method (e.g., credit card, e-wallet, bank transfer)';
COMMENT ON COLUMN payments.payment_detail IS 'Serialized payment method details (JSON or encrypted)';
COMMENT ON COLUMN payments.payment_status IS 'Lifecycle: pending → authorized → captured | cancelled; captured → refunded';
COMMENT ON COLUMN payments.transaction_reference IS 'Reference string for reconciliation (TXN-{order_id})';
COMMENT ON COLUMN payments.idempotency_key IS 'Client-provided key to ensure payment idempotency';
COMMENT ON COLUMN payments.transaction_time IS 'Time of payment transaction';

COMMENT ON COLUMN payment_logs.payment_id IS 'Foreign key to payments table';
COMMENT ON COLUMN payment_logs.status IS 'Status snapshot at the time of log entry';
COMMENT ON COLUMN payment_logs.timestamp IS 'When the status change occurred';

COMMENT ON COLUMN payment_methods.id IS 'Unique identifier for payment method';
COMMENT ON COLUMN payment_methods.method_name IS 'Human-readable method name (e.g., Visa, GCash, BDO)';
COMMENT ON COLUMN payment_methods.provider IS 'Payment provider (e.g., Stripe, Xendit, PayMaya)';
