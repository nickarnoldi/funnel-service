ALTER TABLE abandoned_leads
    ALTER COLUMN country_code TYPE VARCHAR(5),
    ALTER COLUMN region_code TYPE VARCHAR(5);