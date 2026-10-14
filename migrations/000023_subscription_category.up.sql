-- SPEC-0047: a subscription belongs to an expense category, like a purchase.
-- Existing subscriptions stay valid without one; the next save asks for it.
ALTER TABLE subscriptions
    ADD COLUMN category_id UUID REFERENCES categories (id) ON DELETE SET NULL;
CREATE INDEX subscriptions_category_idx ON subscriptions (category_id) WHERE category_id IS NOT NULL;
