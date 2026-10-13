-- SPEC-0042: category icons, owner deletion with reassignment, hiding shared categories.
ALTER TABLE categories ADD COLUMN icon TEXT CHECK (icon IS NULL OR icon IN (
    'shopping-cart','utensils','coffee','car','bus','fuel','home','zap','wifi','smartphone',
    'heart-pulse','pill','shirt','gift','gamepad-2','film','music','book-open','graduation-cap',
    'baby','paw-print','plane','dumbbell','scissors','wrench','credit-card','landmark','receipt',
    'repeat','briefcase','banknote','piggy-bank','sparkles','tag','sofa','cigarette'));

-- Well-known shared categories get a default icon; others fall back to "tag" in the UI.
UPDATE categories SET icon = CASE lower(name)
    WHEN 'продукты' THEN 'shopping-cart' WHEN 'транспорт' THEN 'bus' WHEN 'жильё' THEN 'home'
    WHEN 'жилье' THEN 'home' WHEN 'зарплата' THEN 'banknote' WHEN 'кафе' THEN 'utensils'
    WHEN 'рестораны' THEN 'utensils' WHEN 'связь' THEN 'smartphone' WHEN 'здоровье' THEN 'heart-pulse'
    WHEN 'подписки' THEN 'repeat' WHEN 'развлечения' THEN 'gamepad-2' WHEN 'одежда' THEN 'shirt'
    WHEN 'подарки' THEN 'gift' WHEN 'образование' THEN 'graduation-cap' WHEN 'путешествия' THEN 'plane'
    END
WHERE owner_id IS NULL AND icon IS NULL;

REVOKE UPDATE(name) ON categories FROM grace_financial;
GRANT INSERT(icon), UPDATE(name, icon), DELETE ON categories TO grace_financial;
CREATE POLICY category_delete ON categories FOR DELETE TO grace_financial
    USING (owner_id = grace_owner_id());

CREATE TABLE category_hidden (
    owner_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    hidden_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_id, category_id)
);
ALTER TABLE category_hidden ENABLE ROW LEVEL SECURITY;
ALTER TABLE category_hidden FORCE ROW LEVEL SECURITY;
CREATE POLICY owner_access ON category_hidden TO grace_financial
    USING (owner_id = grace_owner_id()) WITH CHECK (owner_id = grace_owner_id());
GRANT SELECT, INSERT, DELETE ON category_hidden TO grace_financial;
