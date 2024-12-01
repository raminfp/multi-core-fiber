CREATE TABLE request_tracking (
                                  id SERIAL PRIMARY KEY,        -- Add an auto-incrementing primary key
                                  core INT NOT NULL,
                                  request_id BIGINT NOT NULL,   -- Change to BIGINT to accommodate larger values
                                  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
