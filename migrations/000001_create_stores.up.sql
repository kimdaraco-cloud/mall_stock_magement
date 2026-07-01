-- @ai-modified 2026-07-02 create stores table
CREATE TABLE stores (
    id            BIGSERIAL PRIMARY KEY,
    name          VARCHAR(255) NOT NULL,
    unit_number   VARCHAR(50),
    floor         VARCHAR(50),
    category      VARCHAR(100),
    contact_name  VARCHAR(255),
    contact_phone VARCHAR(50),
    contact_email VARCHAR(255),
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
