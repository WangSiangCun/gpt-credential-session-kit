-- Runs before 239_account_market.sql (filename ordering). Only legacy tables
-- need their serial sequences moved out of the new schema's namespace.
DO $$
BEGIN
    IF to_regclass('public.account_market_earnings') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema='public' AND table_name='account_market_earnings'
             AND column_name='uploader_user_id'
       ) THEN
        ALTER SEQUENCE IF EXISTS account_market_earnings_id_seq
            RENAME TO account_market_earnings_legacy_pre239_id_seq;
    END IF;
    IF to_regclass('public.account_market_withdrawals') IS NOT NULL
       AND NOT EXISTS (
           SELECT 1 FROM information_schema.columns
           WHERE table_schema='public' AND table_name='account_market_withdrawals'
             AND column_name='commission_percent'
       ) THEN
        ALTER SEQUENCE IF EXISTS account_market_withdrawals_id_seq
            RENAME TO account_market_withdrawals_legacy_pre239_id_seq;
    END IF;
END $$;
