CREATE TABLE logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level       TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error')),
    message     TEXT NOT NULL,
    source      TEXT NOT NULL,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    request_id  UUID,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_logs_level_created_at ON logs (level, created_at DESC);
CREATE INDEX idx_logs_created_at ON logs (created_at DESC);
CREATE INDEX idx_logs_user_id ON logs (user_id);
CREATE INDEX idx_logs_request_id ON logs (request_id) WHERE request_id IS NOT NULL;
