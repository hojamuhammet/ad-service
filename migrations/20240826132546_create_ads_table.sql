-- +goose Up
CREATE TABLE ads (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    price DECIMAL(10, 2) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE INDEX idx_title ON ads(title);
CREATE INDEX idx_price ON ads(price);
CREATE INDEX idx_created_at ON ads(created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_title ON ads;
DROP INDEX IF EXISTS idx_price ON ads;
DROP INDEX IF EXISTS idx_created_at ON ads;
DROP TABLE IF EXISTS ads;
