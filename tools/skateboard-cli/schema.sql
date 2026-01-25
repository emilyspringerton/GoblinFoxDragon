-- Source [4]
CREATE TABLE IF NOT EXISTS chunks (
    id VARCHAR(64) PRIMARY KEY,
    type ENUM('VISION', 'SETUP', 'ARCHITECTURE', 'FEATURE', 'INCOMPLETE', 'DEAD_TEXT') NOT NULL,
    title VARCHAR(255) NOT NULL,
    source_text TEXT,
    status ENUM('unprocessed', 'planned', 'in_progress', 'done', 'blocked') DEFAULT 'unprocessed',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS actions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    chunk_id VARCHAR(64),
    description TEXT NOT NULL,
    is_completed BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS commits (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    chunk_id VARCHAR(64),
    git_hash VARCHAR(40),
    message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (chunk_id) REFERENCES chunks(id) ON DELETE SET NULL
);
