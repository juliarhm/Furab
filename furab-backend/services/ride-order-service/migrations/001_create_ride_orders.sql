-- Migration: 001_create_ride_orders.sql
-- Creates the ride_orders table for the ride-order-service.
--
-- Run this migration manually or via a migration tool:
--   psql -U furab -d ride_order_service -f migrations/001_create_ride_orders.sql

-- Create ride_orders table
CREATE TABLE IF NOT EXISTS ride_orders (
    id                 VARCHAR(36) PRIMARY KEY,
    user_id            VARCHAR(36) NOT NULL,
    driver_id          VARCHAR(36),
    pickup_lat         DOUBLE PRECISION NOT NULL,
    pickup_lng         DOUBLE PRECISION NOT NULL,
    pickup_address     TEXT NOT NULL,
    dropoff_lat        DOUBLE PRECISION NOT NULL,
    dropoff_lng        DOUBLE PRECISION NOT NULL,
    dropoff_address    TEXT NOT NULL,
    status             VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    fare               DOUBLE PRECISION NOT NULL DEFAULT 0,
    distance           DOUBLE PRECISION NOT NULL DEFAULT 0,
    estimated_duration INTEGER NOT NULL DEFAULT 0,
    created_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_ride_orders_user_id ON ride_orders(user_id);
CREATE INDEX IF NOT EXISTS idx_ride_orders_driver_id ON ride_orders(driver_id);
CREATE INDEX IF NOT EXISTS idx_ride_orders_status ON ride_orders(status);
CREATE INDEX IF NOT EXISTS idx_ride_orders_created_at ON ride_orders(created_at DESC);

-- Add a status check constraint
ALTER TABLE ride_orders 
ADD CONSTRAINT chk_ride_status 
CHECK (status IN ('PENDING', 'ASSIGNED', 'STARTED', 'COMPLETED', 'CANCELLED'));

-- Comment on table
COMMENT ON TABLE ride_orders IS 'Stores all ride order data for the Furab ride-hailing service';
COMMENT ON COLUMN ride_orders.id IS 'UUID primary key';
COMMENT ON COLUMN ride_orders.status IS 'Order lifecycle: PENDING → ASSIGNED → STARTED → COMPLETED | CANCELLED';
COMMENT ON COLUMN ride_orders.fare IS 'Calculated fare in IDR (Rupiah)';
COMMENT ON COLUMN ride_orders.distance IS 'Distance in kilometers';
COMMENT ON COLUMN ride_orders.estimated_duration IS 'Estimated ride duration in minutes';
