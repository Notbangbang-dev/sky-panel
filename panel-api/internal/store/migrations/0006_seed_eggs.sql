-- +goose Up
-- A starter catalog of eggs so a fresh install has something to create
-- servers from immediately. The Minecraft eggs use itzg/minecraft-server,
-- which downloads everything itself from env vars — Startup is left empty
-- so sky-daemon omits Docker's Cmd override and the image's own entrypoint
-- runs. EULA defaults to TRUE (still shown/editable, not silently forced)
-- since these are meant to work the moment a server is created.
INSERT INTO eggs (id, name, category, description, docker_image, startup, stop_command, variables_json) VALUES
('7261099f-ebe4-47df-93fe-18daab5e7aff', 'Paper', 'Minecraft', 'High-performance Minecraft server, a Spigot fork with better plugin API compatibility and much better default performance.', 'itzg/minecraft-server', '', 'stop',
 '[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"Memory","env":"MEMORY","default":"1024M","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"PAPER","user_editable":false}]'),

('63fef0d5-b7bc-48f7-bef4-cc37b896ff35', 'Vanilla Minecraft', 'Minecraft', 'The unmodified official Minecraft Java server, straight from Mojang.', 'itzg/minecraft-server', '', 'stop',
 '[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"Memory","env":"MEMORY","default":"1024M","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"VANILLA","user_editable":false}]'),

('c279ab7a-2618-4fde-9e71-9699ca7f6639', 'Spigot', 'Minecraft', 'The original high-performance Bukkit/CraftBukkit fork most Minecraft plugins target.', 'itzg/minecraft-server', '', 'stop',
 '[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"Memory","env":"MEMORY","default":"1024M","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"SPIGOT","user_editable":false}]'),

('ad1a6e55-b6bb-4b59-900a-d26caf533060', 'Forge (Modded Minecraft)', 'Minecraft', 'Minecraft Forge modded server — drop mod jars in the mods/ folder from the file manager after the first boot.', 'itzg/minecraft-server', '', 'stop',
 '[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"Forge Version","env":"FORGE_VERSION","default":"RECOMMENDED","user_editable":true},{"name":"Memory","env":"MEMORY","default":"2048M","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"FORGE","user_editable":false}]'),

('b3f2b256-7ea5-4ec8-afa4-bf7903452ad9', 'Fabric (Modded Minecraft)', 'Minecraft', 'Fabric modded server — a lighter-weight modding platform than Forge, popular for performance/QoL mods.', 'itzg/minecraft-server', '', 'stop',
 '[{"name":"Version","env":"VERSION","default":"LATEST","user_editable":true},{"name":"Memory","env":"MEMORY","default":"2048M","user_editable":true},{"name":"EULA","env":"EULA","default":"TRUE","user_editable":true},{"name":"Type","env":"TYPE","default":"FABRIC","user_editable":false}]'),

('67a779c3-bc83-44e7-ae37-1a79059290ab', 'BungeeCord Proxy', 'Minecraft', 'BungeeCord proxy for linking multiple Minecraft servers behind a single address.', 'itzg/bungeecord', '', '',
 '[{"name":"Memory","env":"MEMORY","default":"512M","user_editable":true}]'),

('94ac77be-2f8c-4e39-82e8-0d970dd1c0d2', 'Node.js Application', 'Generic', 'Runs a Node.js app from the server''s files. Installs dependencies from package.json on every boot, then runs npm start.', 'node:22-alpine', 'sh -c "npm install --omit=dev 2>/dev/null; npm start"', '',
 '[{"name":"Environment","env":"NODE_ENV","default":"production","user_editable":true}]'),

('03f4fde3-4ddc-4124-b8e6-ddcd9909b8cd', 'Python Application', 'Generic', 'Runs a Python app from the server''s files. Installs dependencies from requirements.txt on every boot, then runs the entrypoint file.', 'python:3.12-slim', 'sh -c "pip install -r requirements.txt 2>/dev/null; python {{APP_ENTRYPOINT}}"', '',
 '[{"name":"Entrypoint file","env":"APP_ENTRYPOINT","default":"app.py","user_editable":true}]'),

('21dd7e05-617a-45d7-a56a-ed5fdde7979b', 'Rust (Facepunch)', 'Survival', 'Facepunch''s Rust dedicated server. The image installs and updates the server itself via SteamCMD on boot.', 'didstopia/rust-server', '', '',
 '[{"name":"Server Name","env":"RUST_SERVER_NAME","default":"My Rust Server","user_editable":true}]'),

('d9daf1ff-e4ad-45c9-b7c1-a7f328558f01', 'Custom Docker Image', 'Generic', 'A blank template for anything not covered above. Edit this egg (Admin > Eggs) to point docker_image at whatever you need before creating servers from it.', 'alpine:latest', '', '',
 '[]');

-- +goose Down
DELETE FROM eggs WHERE id IN (
  '7261099f-ebe4-47df-93fe-18daab5e7aff',
  '63fef0d5-b7bc-48f7-bef4-cc37b896ff35',
  'c279ab7a-2618-4fde-9e71-9699ca7f6639',
  'ad1a6e55-b6bb-4b59-900a-d26caf533060',
  'b3f2b256-7ea5-4ec8-afa4-bf7903452ad9',
  '67a779c3-bc83-44e7-ae37-1a79059290ab',
  '94ac77be-2f8c-4e39-82e8-0d970dd1c0d2',
  '03f4fde3-4ddc-4124-b8e6-ddcd9909b8cd',
  '21dd7e05-617a-45d7-a56a-ed5fdde7979b',
  'd9daf1ff-e4ad-45c9-b7c1-a7f328558f01'
);
