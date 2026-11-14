DROP TRIGGER IF EXISTS stock_movements_immutable ON stock_movements;
DROP FUNCTION IF EXISTS prevent_stock_movement_mutation();
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS inventory_balances;
