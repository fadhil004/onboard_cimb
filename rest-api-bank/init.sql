CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    account_number TEXT,
    account_holder TEXT,
    balance BIGINT
);

CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY,
    from_account_id UUID,
    to_account_id UUID,
    amount BIGINT
);