-- 0004_driver_responsible_manager.sql
--
-- tp.md §8 "equipe sob responsabilidade": a driver may have one
-- responsible manager (the manager's team). It is an attribute for
-- filtering ("my team" views), not a permission boundary — every manager
-- still sees every driver (single company). The service guarantees the
-- referenced account is an active manager; the database guarantees it
-- exists. Deleting a manager account is not an operation (managers are
-- deactivated or anonymized), so no ON DELETE action is needed.

ALTER TABLE driver_profile
    ADD COLUMN manager_user_id uuid REFERENCES app_user (id);

CREATE INDEX driver_profile_manager_idx ON driver_profile (manager_user_id); -- team filters, team_size
