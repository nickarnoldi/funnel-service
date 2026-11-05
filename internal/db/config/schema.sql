/* table for tracking native ids from crms/apis */
CREATE TABLE IF NOT EXISTS external_references (
    id BIGSERIAL PRIMARY KEY,
    platform_id INT NOT NULL DEFAULT 0, /* enum.Platform */
    type VARCHAR(50) NOT NULL,
    external_id BIGINT NOT NULL, /* ID of the respective entity */
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (platform_id, type, external_id)
);

CREATE TABLE IF NOT EXISTS funnels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200),
    url VARCHAR(800),
    description TEXT,
    auto_import BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS funnel_steps (
    id BIGSERIAL PRIMARY KEY,
    funnel_id BIGINT NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    step_index INT NOT NULL DEFAULT 1,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS funnel_fields (
    id BIGSERIAL PRIMARY KEY,
    funnel_id BIGINT NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    funnel_step_id BIGINT REFERENCES funnel_steps(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    do_not_save BOOLEAN NOT NULL DEFAULT false,
    is_required BOOLEAN NOT NULL DEFAULT false,
    is_honeypot BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS funnel_step_conditions (
    id BIGSERIAL PRIMARY KEY,
    funnel_step_id BIGINT NOT NULL REFERENCES funnel_steps(id) ON DELETE CASCADE,
    field VARCHAR(200) NOT NULL,
    operator INT NOT NULL DEFAULT 0, /* enums.ConditionOperator */
    condition VARCHAR(200),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
    id BIGSERIAL PRIMARY KEY,
    reference_id BIGINT REFERENCES external_references(id),
    network_id INT NOT NULL, /* enum.Network */
    name VARCHAR(500) NOT NULL,
    status INT NOT NULL DEFAULT 0, /* enum.Status */
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_categories_network_name ON categories(network_id, name);

CREATE TABLE IF NOT EXISTS advertisers (
    id BIGSERIAL PRIMARY KEY,
    reference_id BIGINT REFERENCES external_references(id),
    network_id INT NOT NULL, /* enum.Network */
    name VARCHAR(500) NOT NULL,
    status INT NOT NULL DEFAULT 0, /* enum.Status */
    default_currency CHAR(3) NOT NULL DEFAULT 'USD',
    attribution_method INT NOT NULL DEFAULT 0, /* enum.AttributionMethod */
    attribution_priority INT NOT NULL DEFAULT 0, /* enum.AttributionPriority */
    email_attibution_method INT NOT NULL DEFAULT 0, /* enum.EmailAttributionMethod */
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS affiliates (
    id BIGSERIAL PRIMARY KEY,
    reference_id BIGINT REFERENCES external_references(id),
    network_id INT NOT NULL, /* enum.Network */
    name VARCHAR(500) NOT NULL,
    status INT NOT NULL DEFAULT 0, /* enum.Status */
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS offers (
    id BIGSERIAL PRIMARY KEY,
    reference_id BIGINT REFERENCES external_references(id),
    network_id INT NOT NULL, /* enum.Network */
    funnel_id BIGINT REFERENCES funnels(id) ON DELETE SET NULL,
    category_id BIGINT NOT NULL REFERENCES categories(id),
    advertiser_id BIGINT NOT NULL REFERENCES advertisers(id),
    name VARCHAR(500) NOT NULL,
    status INT NOT NULL DEFAULT 0, /* enum.OfferStatus */
    thumbnail_url VARCHAR(500),
    visibility INT NOT NULL DEFAULT 0, /* enum.Visibilty */
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS offer_urls (
    id BIGSERIAL PRIMARY KEY,
    reference_id BIGINT REFERENCES external_references(id),
    network_id INT NOT NULL, /* enum.Network */
    funnel_id BIGINT REFERENCES funnels(id) ON DELETE SET NULL,
    offer_id BIGINT NOT NULL REFERENCES offers(id),
    name VARCHAR(500) NOT NULL,
    status INT NOT NULL DEFAULT 0, /* enum.OfferURLStatus */
    destination_url VARCHAR(1000) NOT NULL,
    preview_url VARCHAR(1000),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_offer_urls_destination_url ON offer_urls(destination_url);
CREATE INDEX IF NOT EXISTS idx_offer_urls_funnel_id ON offer_urls(funnel_id);

CREATE TABLE IF NOT EXISTS abandoned_leads (
    id BIGSERIAL PRIMARY KEY,
    offer_id BIGINT NOT NULL REFERENCES offers(id),
    offer_url_id BIGINT NOT NULL REFERENCES offer_urls(id),
    click_id BIGINT REFERENCES raw_clicks(id),
    step_number INT NOT NULL DEFAULT 1,
    transaction_id VARCHAR(40) NOT NULL DEFAULT '',
    ip_address VARCHAR(45) NOT NULL,
    is_bot BOOLEAN NOT NULL DEFAULT false,
    is_mobile BOOLEAN NOT NULL DEFAULT false,
    honeypot_filled BOOLEAN NOT NULL DEFAULT false,
    session_length INT NOT NULL DEFAULT 0,
    jornaya_id VARCHAR(45),
    trustedform_id VARCHAR(45),
    affiliate_id VARCHAR(100),
    campaign_id VARCHAR(100),
    anura_score VARCHAR(15),
    email VARCHAR(100),
    phone_number VARCHAR(20),
    country_code VARCHAR(5),
    region_code VARCHAR(5),
    city VARCHAR(100),
    postal_code VARCHAR(25),
    date_of_birth DATE,
    employment_status SMALLINT,
    gross_monthly_income NUMERIC(10,2),
    landing_page_url VARCHAR(2000) NOT NULL DEFAULT '',
    referrer VARCHAR(2000) NOT NULL DEFAULT '',
    cookies JSONB,
    body JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_abandoned_leads_transaction_id ON abandoned_leads(transaction_id);
CREATE INDEX IF NOT EXISTS idx_abandoned_leads_ip_address ON abandoned_leads(ip_address);
CREATE INDEX IF NOT EXISTS idx_abandoned_leads_created_at ON abandoned_leads(created_at);

CREATE TABLE IF NOT EXISTS abandoned_lead_funnel_step (
    id BIGSERIAL PRIMARY KEY,
    abandoned_lead_id BIGINT NOT NULL UNIQUE REFERENCES abandoned_leads(id),
    funnel_step_id BIGINT NOT NULL REFERENCES funnel_steps(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS raw_clicks (
    id BIGSERIAL PRIMARY KEY,
    offer_id BIGINT NOT NULL REFERENCES offers(id),
    offer_url_id BIGINT REFERENCES offer_urls(id),
    category_id BIGINT NOT NULL REFERENCES categories(id),
    advertiser_id BIGINT NOT NULL REFERENCES advertisers(id),
    affiliate_id BIGINT REFERENCES affiliates(id),
    network_id INT NOT NULL, /* enum.Network */
    transaction_id VARCHAR(100) NOT NULL,
    is_unique BOOLEAN NOT NULL DEFAULT false,
    tracking_url VARCHAR(2000) NOT NULL DEFAULT '',
    source_id VARCHAR(100) NOT NULL DEFAULT '',
    sub1 VARCHAR(500) NOT NULL DEFAULT '',
    sub2 VARCHAR(500) NOT NULL DEFAULT '',
    sub3 VARCHAR(500) NOT NULL DEFAULT '',
    sub4 VARCHAR(500) NOT NULL DEFAULT '',
    sub5 VARCHAR(500) NOT NULL DEFAULT '',
    payout_amount NUMERIC(8,2) NOT NULL DEFAULT 0.00,
    revenue_amount NUMERIC(8,2) NOT NULL DEFAULT 0.00,
    referer TEXT NOT NULL DEFAULT '',
    error_code INT NOT NULL DEFAULT 0,
    error_filter_id VARCHAR(100) NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    is_view_through BOOLEAN NOT NULL DEFAULT false,
    is_async BOOLEAN NOT NULL DEFAULT false,
    server_side_url TEXT NOT NULL DEFAULT '',
    custom_landing_page_id VARCHAR(100) NOT NULL DEFAULT '',
    is_test_mode BOOLEAN NOT NULL DEFAULT false,
    idfa VARCHAR(100) NOT NULL DEFAULT '',
    google_ad_id VARCHAR(100) NOT NULL DEFAULT '',
    creative_id VARCHAR(100) NOT NULL DEFAULT '',
    coupon_code VARCHAR(100) NOT NULL DEFAULT '',
    redirect_method INT NOT NULL DEFAULT 0, /* enum.RedirectMethod */
    is_sdk_click BOOLEAN NOT NULL DEFAULT false,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL,
    geolocation JSONB NOT NULL DEFAULT '{}',
    device_info JSONB NOT NULL DEFAULT '{}',
    traffic_source INT NOT NULL DEFAULT 0, /* enum.TrafficSource */
    is_bot BOOLEAN NOT NULL DEFAULT false,
    is_mobile BOOLEAN NOT NULL DEFAULT false,
    is_tablet BOOLEAN NOT NULL DEFAULT false,
    device_type INT NOT NULL DEFAULT 0, /* enum.DeviceType */
    language_code VARCHAR(5),
    country_code VARCHAR(5),
    region_code VARCHAR(5),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_raw_clicks_unique ON raw_clicks(network_id, transaction_id);
CREATE INDEX IF NOT EXISTS idx_raw_clicks_transaction_id ON raw_clicks(transaction_id);
CREATE INDEX IF NOT EXISTS idx_raw_clicks_created_at ON raw_clicks(created_at);

CREATE TABLE IF NOT EXISTS raw_conversions (
    id BIGSERIAL PRIMARY KEY,
    click_id BIGINT REFERENCES raw_clicks(id),
    offer_id BIGINT REFERENCES offers(id),
    offer_url_id BIGINT REFERENCES offer_urls(id),
    network_id INT NOT NULL, /* enum.Network */
    conversion_id VARCHAR(100) NOT NULL,
    transaction_id VARCHAR(100) NOT NULL,
    status INT NOT NULL DEFAULT 0, /* enum.ConversionStatus */
    payout_model INT, /* enum.PayoutModel */
    payout_amount NUMERIC(8,2) NOT NULL DEFAULT 0.00,
    revenue_model INT, /* enum.RevenueModel */
    revenue_amount NUMERIC(8,2) NOT NULL DEFAULT 0.00,
    ip_address VARCHAR(45) NOT NULL,
    user_agent TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_raw_conversions_network_id ON raw_conversions(network_id);
CREATE INDEX IF NOT EXISTS idx_raw_conversions_conversion_id ON raw_conversions(conversion_id);
CREATE INDEX IF NOT EXISTS idx_raw_conversions_transaction_id ON raw_conversions(transaction_id);
CREATE INDEX IF NOT EXISTS idx_raw_conversions_network_conversion ON raw_conversions(network_id, conversion_id);
CREATE INDEX IF NOT EXISTS idx_raw_conversions_network_transaction ON raw_conversions(network_id, transaction_id);
CREATE INDEX IF NOT EXISTS idx_raw_conversions_created_at ON raw_conversions(created_at);

CREATE TABLE IF NOT EXISTS stats_hourly (
    id BIGSERIAL PRIMARY KEY,
    slice_hour TIMESTAMP NOT NULL,
    offer_id BIGINT NOT NULL REFERENCES offers(id) ON DELETE CASCADE,
    offer_url_id BIGINT NOT NULL REFERENCES offer_urls(id) ON DELETE CASCADE,
    category_id BIGINT NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    advertiser_id BIGINT NOT NULL REFERENCES advertisers(id) ON DELETE CASCADE,
    affiliate_id BIGINT REFERENCES affiliates(id) ON DELETE CASCADE,
    traffic_source INT NOT NULL DEFAULT 0, /* enum.TrafficSource */
    is_unique BOOLEAN NOT NULL DEFAULT false,
    is_bot BOOLEAN NOT NULL DEFAULT false,
    is_mobile BOOLEAN NOT NULL DEFAULT false,
    is_tablet BOOLEAN NOT NULL DEFAULT false,
    device_type INT NOT NULL DEFAULT 0, /* enum.DeviceType */
    language_code VARCHAR(5),
    country_code VARCHAR(5),
    region_code VARCHAR(5),
    clicks INT NOT NULL DEFAULT 0,
    conversions INT NOT NULL DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_stats_hourly_hour_offer ON stats_hourly (slice_hour, offer_id, offer_url_id);
CREATE INDEX IF NOT EXISTS idx_stats_hourly_hour_category_advertiser ON stats_hourly (slice_hour, category_id, advertiser_id);
CREATE INDEX IF NOT EXISTS idx_stats_hourly_hour_affiliate_traffic ON stats_hourly (slice_hour, affiliate_id, traffic_source, is_unique, is_bot);
CREATE INDEX IF NOT EXISTS idx_stats_hourly_hour_device_geo ON stats_hourly (slice_hour, is_mobile, is_tablet, device_type, country_code, region_code, language_code);