-- +goose Up

-- Domain model (aggregate root + entities)
--
-- domain_event
-- outbox
-- currency
--   exchange_rate
-- product_group
--   product_group_weight
-- product
-- threshold_group
--   threshold_group_limit
-- cluster
-- reseller
--   billing
--     billing_item
-- tier_discount
-- tiering

-- domain_event

CREATE TABLE IF NOT EXISTS public.domain_event
(
    -- The id field is required to order events. occurred_at isn't guaranteed to
    -- be unique at high insertion rates and when NTP may be adjusting time.
    -- Also, timers on different platforms may have different resolutions.
    -- Relying on unique occurred_at would cause intermittend insertion
    -- failures.
    --
    -- Unlike with DDD entities, where the application must control the uuid,
    -- the id of a domain event is never used as a foreign key. The application
    -- doesn't need to know the id of the just inserted events.
    --
    -- In a distributed system time of occurrence is subjective, but time of
    -- insertion is absolute. So even if multiple instances of application is
    -- running, the auto-incrementing id guarantees cross-instance ordering.
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1 ),
    aggregate_id uuid NOT NULL,
    type character varying(50) COLLATE pg_catalog."default" NOT NULL,
    payload jsonb NOT NULL,
    version int NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    CONSTRAINT pk_domain_event_id PRIMARY KEY (id)
);

ALTER TABLE IF EXISTS public.domain_event
    OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_domain_event_aggregate_id
    ON public.domain_event USING btree
    (aggregate_id ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

-- outbox_event

CREATE TABLE IF NOT EXISTS public.outbox_event
(
    id bigint NOT NULL GENERATED ALWAYS AS IDENTITY ( INCREMENT 1 START 1 MINVALUE 1 MAXVALUE 9223372036854775807 CACHE 1 ),
    type character varying(50) COLLATE pg_catalog."default" NOT NULL,
    payload jsonb NOT NULL,
    occurred_at timestamp with time zone NOT NULL,
    processed_at timestamp with time zone,
    CONSTRAINT pk_outbox_event_id PRIMARY KEY (id)
);

ALTER TABLE IF EXISTS public.outbox_event
    OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_outbox_event_processed_at_occurred_at
    ON public.outbox_event USING btree
    (processed_at ASC NULLS LAST, occurred_at ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

-- currency

CREATE TABLE IF NOT EXISTS public.currency
(
    id uuid NOT NULL,
    code character varying(3) COLLATE pg_catalog."default",
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_currency_id PRIMARY KEY (id),
    CONSTRAINT uq_currency_code UNIQUE (code)
);

ALTER TABLE IF EXISTS public.currency
    OWNER to postgres;

-- exchange_rate

CREATE TABLE IF NOT EXISTS public.exchange_rate
(
    id uuid NOT NULL,
    currency_id uuid NOT NULL,
    rate numeric(10,6) NOT NULL,
    "from" date NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_exchange_rate_id PRIMARY KEY (id),
    CONSTRAINT uq_exchange_rate_from UNIQUE ("from"),
    CONSTRAINT fk_currency_id FOREIGN KEY (currency_id)
        REFERENCES public.currency (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.exchange_rate
    OWNER to postgres;

-- product_group

CREATE TABLE IF NOT EXISTS public.product_group
(
    id uuid NOT NULL,
    code character varying(10) COLLATE pg_catalog."default" NOT NULL,
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_product_group_id PRIMARY KEY (id),
    CONSTRAINT uq_product_group_code UNIQUE (code)
);

-- product_group_weight

CREATE TABLE IF NOT EXISTS public.product_group_weight
(
    id uuid NOT NULL,
    product_group_id uuid NOT NULL,
    percentage numeric(5,2) NOT NULL,
    "from" date NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_product_group_weight_id PRIMARY KEY (id),
    CONSTRAINT uq_product_group_from UNIQUE ("from"),
    CONSTRAINT fk_product_group_weight_product_group_id FOREIGN KEY (product_group_id)
        REFERENCES public.product_group (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.product_group_weight
    OWNER to postgres;

ALTER TABLE IF EXISTS public.product_group
    OWNER to postgres;

-- product

CREATE TABLE IF NOT EXISTS public.product
(
    id uuid NOT NULL,
    product_group_id uuid NOT NULL,
    code character varying(10) COLLATE pg_catalog."default" NOT NULL,
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_product_id PRIMARY KEY (id),
    CONSTRAINT uq_product_code UNIQUE (code),
    CONSTRAINT fk_product_group_id FOREIGN KEY (product_group_id)
        REFERENCES public.product_group (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.product
    OWNER to postgres;

-- threshold_group

CREATE TABLE IF NOT EXISTS public.threshold_group
(
    id uuid NOT NULL,
    country_code character varying(2) COLLATE pg_catalog."default" NOT NULL,
    currency_code character varying(3) COLLATE pg_catalog."default" NOT NULL,
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_threshold_group_id PRIMARY KEY (id),
    CONSTRAINT uq_threshold_group_country_code UNIQUE (country_code)
);

ALTER TABLE IF EXISTS public.threshold_group
    OWNER to postgres;

-- threshold_group_limit

CREATE TABLE IF NOT EXISTS public.threshold_group_limit
(
    id uuid NOT NULL,
    threshold_group_id uuid NOT NULL,
    currency_code character varying(3) COLLATE pg_catalog."default" NOT NULL,
    minimum_authorized_net_revenue numeric(12,2) NOT NULL,
    minimum_advanced_net_revenue numeric(12,2) NOT NULL,
    minimum_premier_net_revenue numeric(12,2) NOT NULL,
    "from" date NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_threshold_group_limit_id PRIMARY KEY (id),
    CONSTRAINT uq_threshold_group_limit_from UNIQUE ("from"),
    CONSTRAINT fk_threshold_group_id FOREIGN KEY (threshold_group_id)
        REFERENCES public.threshold_group (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.threshold_group_limit
    OWNER to postgres;

-- cluster

CREATE TABLE IF NOT EXISTS public.cluster
(
    id uuid NOT NULL,
    external_id uuid NOT NULL,
    threshold_group_id uuid NOT NULL,
    calculated_reseller_tier_minimum character varying(10) COLLATE pg_catalog."default",
    calculated_reseller_tier_year_to_date character varying(10) COLLATE pg_catalog."default",
    calculated_reseller_tier_projected character varying(10) COLLATE pg_catalog."default",
    calculated_net_revenue_year_to_date numeric(12,2),
    calculated_net_revenue_last_year numeric(12,2),
    last_tiered_at date,
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_cluster_id PRIMARY KEY (id),
    CONSTRAINT uq_cluster_external_id UNIQUE (external_id),
    CONSTRAINT fk_threshold_group_id FOREIGN KEY (threshold_group_id)
        REFERENCES public.threshold_group (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.cluster
    OWNER to postgres;

-- reseller

CREATE TABLE IF NOT EXISTS public.reseller
(
    id uuid NOT NULL,
    external_id uuid NOT NULL,
    cluster_id uuid,
    country_code character varying(2) COLLATE pg_catalog."default" NOT NULL,
    currency_code character varying(3) COLLATE pg_catalog."default" NOT NULL,
    role character varying(6) COLLATE pg_catalog."default" NOT NULL,
    enrolled_at date NOT NULL,
    calculated_net_revenue_year_to_date numeric(12,2),
    calculated_net_revenue_last_year numeric(12,2),
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_reseller_id PRIMARY KEY (id),
    CONSTRAINT uq_reseller_external_id UNIQUE (external_id),
    CONSTRAINT fk_cluster_id FOREIGN KEY (cluster_id)
        REFERENCES public.cluster (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.reseller
    OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_reseller_enrolled_at
    ON public.reseller USING btree
    (enrolled_at ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

CREATE INDEX IF NOT EXISTS idx_reseller_role
    ON public.reseller USING btree
    (role COLLATE pg_catalog."default" ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

-- billing

CREATE TABLE IF NOT EXISTS public.billing
(
    id uuid NOT NULL,
    reseller_id uuid NULL,
    document_number character varying(10) COLLATE pg_catalog."default" NOT NULL,
    booked_at date NOT NULL,
    billing_kind character varying(10) COLLATE pg_catalog."default" NOT NULL,
    currency_code character varying(3) COLLATE pg_catalog."default" NOT NULL,
    calculated_net_revenue numeric(12,2) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_billing_id PRIMARY KEY (id),
    CONSTRAINT uq_billing_document_number UNIQUE (document_number),
    CONSTRAINT fk_billing_reseller_id FOREIGN KEY (reseller_id)
        REFERENCES public.reseller (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.billing
    OWNER to postgres;

CREATE INDEX IF NOT EXISTS idx_billing_booked_at
    ON public.billing USING btree
    (booked_at ASC NULLS LAST)
    WITH (fillfactor=100, deduplicate_items=True)
    TABLESPACE pg_default;

-- billing_item

CREATE TABLE IF NOT EXISTS public.billing_item
(
    id uuid NOT NULL,
    billing_id uuid NOT NULL,
    product_id uuid NOT NULL,
    currency_code character varying(3) COLLATE pg_catalog."default" NOT NULL,
    gross_revenue numeric(12,2) NOT NULL,
    calculated_net_revenue numeric(12,2) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_billing_item_id PRIMARY KEY (id),
    CONSTRAINT fk_product_id FOREIGN KEY (product_id)
        REFERENCES public.product (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION,
    CONSTRAINT fk_billing_id FOREIGN KEY (billing_id)
        REFERENCES public.billing (id) MATCH SIMPLE
        ON UPDATE NO ACTION
        ON DELETE NO ACTION
);

ALTER TABLE IF EXISTS public.billing_item
    OWNER to postgres;

-- tier_discount

CREATE TABLE IF NOT EXISTS public.tier_discount
(
    id uuid NOT NULL,
    authorized_percentage numeric(5,2) NOT NULL,
    advanced_percentage numeric(5,2) NOT NULL,
    premier_percentage numeric(5,2) NOT NULL,
    "from" date NOT NULL,
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_tier_discount_id PRIMARY KEY (id),
    CONSTRAINT uq_tier_discount_from UNIQUE ("from")
);

ALTER TABLE IF EXISTS public.tier_discount
    OWNER to postgres;

-- tiering

CREATE TABLE IF NOT EXISTS public.tiering
(
    id uuid NOT NULL,
    tier_at date NOT NULL,
    start timestamp with time zone NOT NULL,
    "end" timestamp with time zone NOT NULL,
    version int NOT NULL,
    created_at timestamp with time zone NOT NULL,
    updated_at timestamp with time zone,
    CONSTRAINT pk_tiering_id PRIMARY KEY (id),
    CONSTRAINT uq_tiering_tier_at UNIQUE (tier_at)
);

ALTER TABLE IF EXISTS public.tiering
    OWNER to postgres;

-- +goose Down
-- Intentionally left blank.