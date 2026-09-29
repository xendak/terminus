-- 0002_audit_entity_id_text.sql
--
-- audit_log.entity_id must hold parameter keys (text PKs, e.g.
-- 'fuel_price_brl') as well as uuid strings for route_stop/route rows.
-- The uuid typing was a T2 spec bug, surfaced by UpdateParam's audit row
-- (update_param, T4). docs/spec/data-model.md is corrected in the same
-- commit; 0001 itself is immutable (fixes are new migrations).

ALTER TABLE audit_log ALTER COLUMN entity_id TYPE text;
