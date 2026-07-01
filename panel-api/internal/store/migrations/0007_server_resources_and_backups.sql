-- +goose Up
-- Per-server resource limits and backup scheduling.
ALTER TABLE servers ADD COLUMN cpu_limit INTEGER NOT NULL DEFAULT 0;              -- percent of one core; 0 = unlimited
ALTER TABLE servers ADD COLUMN backup_interval_hours INTEGER NOT NULL DEFAULT 0; -- 0 = no scheduled backups
ALTER TABLE servers ADD COLUMN last_backup_at TIMESTAMP;                          -- NULL until the first backup runs

-- Drop the redundant per-egg "Memory" variable from the itzg-based eggs: the
-- container's memory is now driven solely by the server's Memory (MB) limit,
-- which the panel injects as the MEMORY env var. Leaves each egg's other
-- variables intact.
UPDATE eggs SET variables_json='[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"PAPER","user_editable":false}]' WHERE id='7261099f-ebe4-47df-93fe-18daab5e7aff';
UPDATE eggs SET variables_json='[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"VANILLA","user_editable":false}]' WHERE id='63fef0d5-b7bc-48f7-bef4-cc37b896ff35';
UPDATE eggs SET variables_json='[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"SPIGOT","user_editable":false}]' WHERE id='c279ab7a-2618-4fde-9e71-9699ca7f6639';
UPDATE eggs SET variables_json='[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"Forge Version","env":"FORGE_VERSION","default":"RECOMMENDED","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"FORGE","user_editable":false}]' WHERE id='ad1a6e55-b6bb-4b59-900a-d26caf533060';
UPDATE eggs SET variables_json='[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"FABRIC","user_editable":false}]' WHERE id='b3f2b256-7ea5-4ec8-afa4-bf7903452ad9';
UPDATE eggs SET variables_json='[]' WHERE id='67a779c3-bc83-44e7-ae37-1a79059290ab';

-- +goose Down
ALTER TABLE servers DROP COLUMN last_backup_at;
ALTER TABLE servers DROP COLUMN backup_interval_hours;
ALTER TABLE servers DROP COLUMN cpu_limit;
