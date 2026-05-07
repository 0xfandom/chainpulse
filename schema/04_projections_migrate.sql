-- Idempotent migration that backfills projections onto existing
-- token_transfers and defi_events tables. New installs already have
-- them via the inline definitions in 01_token_transfers.sql /
-- 02_defi_events.sql. This file exists so an operator running a
-- pre-existing cluster can pick up the new wallet-centric read paths
-- without a full schema rebuild.
--
-- Safe to re-run: ADD PROJECTION uses IF NOT EXISTS, MATERIALIZE is a
-- no-op when there is nothing left to materialize.

ALTER TABLE token_transfers
    ADD PROJECTION IF NOT EXISTS p_from_time
    (
        SELECT *
        ORDER BY (from_addr, timestamp)
    );

ALTER TABLE token_transfers
    ADD PROJECTION IF NOT EXISTS p_to_time
    (
        SELECT *
        ORDER BY (to_addr, timestamp)
    );

ALTER TABLE token_transfers MATERIALIZE PROJECTION p_from_time;
ALTER TABLE token_transfers MATERIALIZE PROJECTION p_to_time;

ALTER TABLE defi_events
    ADD PROJECTION IF NOT EXISTS p_user_time
    (
        SELECT *
        ORDER BY (user_addr, timestamp)
    );

ALTER TABLE defi_events MATERIALIZE PROJECTION p_user_time;
