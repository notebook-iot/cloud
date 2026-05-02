-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS sensor_data (
    timestamp           TIMESTAMPTZ       NOT NULL,
    device_id           TEXT              NOT NULL,
    api_key             TEXT,
    json_payload        TEXT, -- 'json' is often a reserved word, renamed for clarity
    wifi_network_name   TEXT,
    up_time             BIGINT,
    latency             BIGINT,
    mac_address         TEXT,
    temperature         DOUBLE PRECISION
);

-- Convert to hypertable using the 'timestamp' column
SELECT create_hypertable('sensor_data', 'timestamp', if_not_exists => TRUE);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sensor_data;
-- +goose StatementEnd